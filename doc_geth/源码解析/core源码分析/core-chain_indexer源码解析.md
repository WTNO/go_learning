# chain_indexer 区块链索引
## chain_indexer.go 源码解析
chain_indexer 顾名思义， 就是用来给区块链创建索引的功能。 之前在eth协议的时候，介绍过BloomIndexer的功能，其实BloomIndexer是chain_indexer的一个特殊的实现， 可以理解为派生类， 主要的功能其实是在chain_indexer这里面实现的。虽说是派生类，但是chain_indexer其实就只被BloomIndexer使用。也就是给区块链的布隆过滤器创建了索引，以便快速的响应用户的日志搜索功能。 下面就来分析这部分的代码。

### 数据结构
```go
// ChainIndexerBackend 定义了在后台处理链段并将段结果写入数据库所需的方法。
// 这些方法可以用于创建过滤器布隆或 CHTs(紧凑默克尔树)。
type ChainIndexerBackend interface {
    // Reset 会启动对新的链段的处理，可能会终止任何部分完成的操作（在重新组织的情况下）。
    Reset(ctx context.Context, section uint64, prevHead common.Hash) error
    
    // Process（处理）会处理链段中的下一个区块头。调用者将确保区块头的顺序是连续的。
    Process(ctx context.Context, header *types.Header) error
    
    // Commit（提交）会完成区块段的元数据，并将其存储到数据库中。
    Commit() error
    
    // Prune（修剪）会删除早于给定阈值的链索引。
    Prune(threshold uint64) error
}

// ChainIndexer（链索引器）对规范链的等大小段进行后处理工作（如BlooomBits和CHT结构）。
// 通过在goroutine中启动ChainHeadEventLoop，ChainIndexer通过事件系统与区块链连接。
// 
// 可以添加更多的子ChainIndexers，它们使用父节索引器的输出。
// 这些子索引器只在整个段完成后或可能影响已完成段的回滚情况下才会收到新的头部通知。
type ChainIndexer struct {
    chainDb  ethdb.Database      // 链数据库用于索引数据，
    indexDb  ethdb.Database      // 带前缀的数据库表视图用于将索引元数据写入，
    backend  ChainIndexerBackend // 后台处理器生成索引数据内容，
    children []*ChainIndexer     // 子索引器将链更新级联到。
    
    active    atomic.Bool     // 标记事件循环是否已启动，
    update    chan struct{}   // 通知通道，用于处理区块头，
    quit      chan chan error // 退出通道，用于停止正在运行的goroutine。
    ctx       context.Context
    ctxCancel func()
    
    sectionSize uint64 // 单个链段要处理的区块数量，
    confirmsReq uint64 // 在处理已完成的段之前的确认数。
    
    storedSections uint64 // 成功索引到数据库中的段数，
    knownSections  uint64 // 已知完整（按区块计算）的段数，
    cascadedHead   uint64 // 级联到子索引器的上一个已完成段的区块编号。
    
    checkpointSections uint64      // 检查点覆盖的段数，
    checkpointHead     common.Hash // 属于检查点的段头，
    
    throttling time.Duration // Disk t限制资源使用，以防止重度升级占用资源。
    
    log  log.Logger
    lock sync.Mutex
}
```

构造函数NewChainIndexer
```go
// bloom_indexer.go
const (
    // bloomThrottling（布隆节流）是在处理两个连续的索引段之间等待的时间。
	// 在链升级期间，它对于防止磁盘过载非常有用。
    bloomThrottling = 100 * time.Millisecond
)

// NewBloomIndexer返回一个链索引器，用于为规范链生成布隆比特数据，以进行快速的日志过滤。
func NewBloomIndexer(db ethdb.Database, size, confirms uint64) *ChainIndexer {
    backend := &BloomIndexer{
    db:   db,
    size: size,
    }
    table := rawdb.NewTable(db, string(rawdb.BloomBitsIndexPrefix))
    
    return NewChainIndexer(db, table, backend, size, confirms, bloomThrottling, "bloombits")
}
```

```go
// chain_indexer.go
// NewChainIndexer会创建一个新的链索引器，以在经过一定数量的确认后，
// 对给定大小的链段进行后台处理。节流参数可能会用于防止数据库过载。
func NewChainIndexer(chainDb ethdb.Database, indexDb ethdb.Database, backend ChainIndexerBackend, section, confirm uint64, throttling time.Duration, kind string) *ChainIndexer {
	c := &ChainIndexer{
		chainDb:     chainDb,
		indexDb:     indexDb,
		backend:     backend,
		update:      make(chan struct{}, 1),
		quit:        make(chan chan error),
		sectionSize: section,
		confirmsReq: confirm,
		throttling:  throttling,
		log:         log.New("type", kind),
	}
	// 初始化与数据库相关的字段并启动更新程序。
	c.loadValidSections()
	c.ctx, c.ctxCancel = context.WithCancel(context.Background())

	go c.updateLoop()

	return c
}
```

loadValidSections用来从数据库里面加载我们之前的处理信息， storedSections表示我们已经处理到哪里了。
```go
// loadValidSections reads the number of valid sections from the index database
// and caches is into the local state.
func (c *ChainIndexer) loadValidSections() {
	data, _ := c.indexDb.Get([]byte("count"))
	if len(data) == 8 {
		c.storedSections = binary.BigEndian.Uint64(data)
	}
}
```

updateLoop是主要的事件循环，用于调用backend来处理区块链section，这个需要注意的是，所有的主索引节点和所有的 child indexer 都会启动这个goroutine 方法。
```go
// updateLoop is the main event loop of the indexer which pushes chain segments
// down into the processing backend.
func (c *ChainIndexer) updateLoop() {
	var (
		updating bool
		updated  time.Time
	)

	for {
		select {
		case errc := <-c.quit:
			// Chain indexer terminating, report no failure and abort
			errc <- nil
			return

		case <-c.update:
			// Section headers completed (or rolled back), update the index
			c.lock.Lock()
			if c.knownSections > c.storedSections {
				// Periodically print an upgrade log message to the user
				if time.Since(updated) > 8*time.Second {
					if c.knownSections > c.storedSections+1 {
						updating = true
						c.log.Info("Upgrading chain index", "percentage", c.storedSections*100/c.knownSections)
					}
					updated = time.Now()
				}
				// Cache the current section count and head to allow unlocking the mutex
				c.verifyLastHead()
				section := c.storedSections
				var oldHead common.Hash
				if section > 0 {
					oldHead = c.SectionHead(section - 1)
				}
				// Process the newly defined section in the background
				c.lock.Unlock()
				newHead, err := c.processSection(section, oldHead)
				if err != nil {
					select {
					case <-c.ctx.Done():
						<-c.quit <- nil
						return
					default:
					}
					c.log.Error("Section processing failed", "error", err)
				}
				c.lock.Lock()

				// If processing succeeded and no reorgs occurred, mark the section completed
				if err == nil && (section == 0 || oldHead == c.SectionHead(section-1)) {
					c.setSectionHead(section, newHead)
					c.setValidSections(section + 1)
					if c.storedSections == c.knownSections && updating {
						updating = false
						c.log.Info("Finished upgrading chain index")
					}
					c.cascadedHead = c.storedSections*c.sectionSize - 1
					for _, child := range c.children {
						c.log.Trace("Cascading chain index update", "head", c.cascadedHead)
						child.newHead(c.cascadedHead, false)
					}
				} else {
					// If processing failed, don't retry until further notification
					c.log.Debug("Chain index processing failed", "section", section, "err", err)
					c.verifyLastHead()
					c.knownSections = c.storedSections
				}
			}
			// If there are still further sections to process, reschedule
			if c.knownSections > c.storedSections {
				time.AfterFunc(c.throttling, func() {
					select {
					case c.update <- struct{}{}:
					default:
					}
				})
			}
			c.lock.Unlock()
		}
	}
}
```

Start方法在eth协议启动的时候被调用,这个方法接收两个参数，一个是当前的区块头，一个是事件订阅器，通过这个订阅器可以获取区块链的改变信息。
```go
// backend.go
func New(stack *node.Node, config *ethconfig.Config) (*Ethereum, error) {
    ...
	eth.bloomIndexer.Start(eth.blockchain)
    ...
}

// Start creates a goroutine to feed chain head events into the indexer for
// cascading background processing. Children do not need to be started, they
// are notified about new events by their parents.
func (c *ChainIndexer) Start(chain ChainIndexerChain) {
	events := make(chan ChainHeadEvent, 10)
	sub := chain.SubscribeChainHeadEvent(events)

	go c.eventLoop(chain.CurrentHeader(), events, sub)
}

// eventLoop is a secondary - optional - event loop of the indexer which is only
// started for the outermost indexer to push chain head events into a processing
// queue.
func (c *ChainIndexer) eventLoop(currentHeader *types.Header, events chan ChainHeadEvent, sub event.Subscription) {
    // Mark the chain indexer as active, requiring an additional teardown
    c.active.Store(true)
    
    defer sub.Unsubscribe()
    
    // Fire the initial new head event to start any outstanding processing
    c.newHead(currentHeader.Number.Uint64(), false)
    
    var (
        prevHeader = currentHeader
        prevHash   = currentHeader.Hash()
    )
    for {
        select {
        case errc := <-c.quit:
            // Chain indexer terminating, report no failure and abort
            errc <- nil
            return
        
        case ev, ok := <-events:
            // Received a new event, ensure it's not nil (closing) and update
            if !ok {
                errc := <-c.quit
                errc <- nil
                return
            }
            header := ev.Block.Header()
            if header.ParentHash != prevHash {
                // Reorg to the common ancestor if needed (might not exist in light sync mode, skip reorg then)
                // TODO(karalabe, zsfelfoldi): This seems a bit brittle, can we detect this case explicitly?
                
                if rawdb.ReadCanonicalHash(c.chainDb, prevHeader.Number.Uint64()) != prevHash {
                    if h := rawdb.FindCommonAncestor(c.chainDb, prevHeader, header); h != nil {
                        c.newHead(h.Number.Uint64(), true)
                    }
                }
            }
            c.newHead(header.Number.Uint64(), false)
            
            prevHeader, prevHash = header, header.Hash()
        }
    }
}
```















































