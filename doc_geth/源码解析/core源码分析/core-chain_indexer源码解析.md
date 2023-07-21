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
	// 可以看到indexDb和chainDb实际是同一个数据库， 但是indexDb的每个key前面附加了一个BloomBitsIndexPrefix的前缀。
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
// loadValidSections从索引数据库中读取有效部分的数量，并将其缓存到本地状态中。
func (c *ChainIndexer) loadValidSections() {
	data, _ := c.indexDb.Get([]byte("count"))
	if len(data) == 8 {
		c.storedSections = binary.BigEndian.Uint64(data)
	}
}
```

updateLoop是主要的事件循环，用于调用backend来处理区块链section，这个需要注意的是，所有的主索引节点和所有的 child indexer 都会启动这个goroutine 方法。
```go
// updateLoop是索引器的主要事件循环，它将链段推送到处理后端。
func (c *ChainIndexer) updateLoop() {
	var (
		updating bool
		updated  time.Time
	)

	for {
		select {
		case errc := <-c.quit:
			// 链索引器终止，报告没有失败并中止。
			errc <- nil
			return

		case <-c.update: // 当需要使用backend处理的时候，其他goroutine会往这个channel上面发送消息
			// 部分标题已完成（或回滚），更新索引。
			c.lock.Lock()
			if c.knownSections > c.storedSections { // 如果当前以知的Section 大于已经存储的Section
				// 定期向用户打印升级日志消息。
				if time.Since(updated) > 8*time.Second {
					if c.knownSections > c.storedSections+1 {
						updating = true
						c.log.Info("Upgrading chain index", "percentage", c.storedSections*100/c.knownSections)
					}
					updated = time.Now()
				}
				// 缓存当前部分计数和头部，以允许解锁互斥锁。
				c.verifyLastHead()
				section := c.storedSections
				var oldHead common.Hash
				if section > 0 { // section - 1 代表section的下标是从0开始的。
					// sectionHead用来获取section的最后一个区块的hash值。
					oldHead = c.SectionHead(section - 1)
				}
				// 在后台处理新定义的部分。
				c.lock.Unlock()
				// 处理 返回新的section的最后一个区块的hash值
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

				// 如果处理成功且没有发生重组，则标记该部分已完成。
				if err == nil && (section == 0 || oldHead == c.SectionHead(section-1)) {
					c.setSectionHead(section, newHead)  // 更新数据库的状态
					c.setValidSections(section + 1)     // 更新数据库状态
					if c.storedSections == c.knownSections && updating {
						updating = false
						c.log.Info("Finished upgrading chain index")
					}
					// cascadedHead 是更新后的section的最后一个区块的高度
					c.cascadedHead = c.storedSections*c.sectionSize - 1
					for _, child := range c.children {
						c.log.Trace("Cascading chain index update", "head", c.cascadedHead)
						child.newHead(c.cascadedHead, false)
					}
				} else {
					// 如果处理失败，那么在有新的通知之前不会重试。
					c.log.Debug("Chain index processing failed", "section", section, "err", err)
					c.verifyLastHead()
					c.knownSections = c.storedSections
				}
			}
			// 如果还有section等待处理，那么等待throttling时间再处理。避免磁盘过载。
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

// Start函数创建一个goroutine，将链头事件传递给索引器，以进行级联的后台处理。
// 子节点不需要启动，它们会通过父节点收到新事件的通知。
func (c *ChainIndexer) Start(chain ChainIndexerChain) {
	events := make(chan ChainHeadEvent, 10)
	sub := chain.SubscribeChainHeadEvent(events)

	go c.eventLoop(chain.CurrentHeader(), events, sub)
}

// eventLoop是索引器的次要（可选）事件循环，仅在最外层的索引器中启动，将链头事件推送到处理队列中。
// eventLoop 循环只会在最外面的索引节点被调用。 所有的Child indexer不会被启动这个方法。 
func (c *ChainIndexer) eventLoop(currentHeader *types.Header, events chan ChainHeadEvent, sub event.Subscription) {
    // 标记链索引器为活动状态，需要额外的拆除操作。
    c.active.Store(true)
    
    defer sub.Unsubscribe()
    
    // 触发初始的新头事件，以启动任何未完成的处理。
	// 设置我们的起始的区块高度，用来触发之前未完成的操作。
    c.newHead(currentHeader.Number.Uint64(), false)
    
    var (
        prevHeader = currentHeader
        prevHash   = currentHeader.Hash()
    )
    for {
        select {
        case errc := <-c.quit:
            // 链索引器终止，报告没有失败并中止。
            errc <- nil
            return
        
        case ev, ok := <-events:
            // 收到一个新事件，请确保它不是空（关闭）并进行更新。
            if !ok {
                errc := <-c.quit
                errc <- nil
                return
            }
            header := ev.Block.Header()
            if header.ParentHash != prevHash {
				// 如果出现了分叉，那么我们首先找到公共祖先，从公共祖先之后的索引需要重建。 
                // 如果需要，在必要时重新组织到公共祖先（在轻量同步模式下可能不存在，那么跳过重新组织）。
                // TODO(karalabe, zsfelfoldi): This seems a bit brittle, can we detect this case explicitly?
                
                if rawdb.ReadCanonicalHash(c.chainDb, prevHeader.Number.Uint64()) != prevHash {
                    if h := rawdb.FindCommonAncestor(c.chainDb, prevHeader, header); h != nil {
                        c.newHead(h.Number.Uint64(), true)
                    }
                }
            }
			// 设置新的head
            c.newHead(header.Number.Uint64(), false)
            
            prevHeader, prevHash = header, header.Hash()
        }
    }
}
```

newHead方法,通知indexer新的区块链头，或者是需要重建索引，newHead方法会触发
```go
// newHead函数通知索引器有关新的链头和/或重新组织的信息。
func (c *ChainIndexer) newHead(head uint64, reorg bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 如果发生了重新组织，使该点之前的所有部分失效。
	if reorg {
		// 将已知的部分编号回滚到重新组织点。
		known := (head + 1) / c.sectionSize
		stored := known
		if known < c.checkpointSections {
			known = 0
		}
		if stored < c.checkpointSections {
			stored = c.checkpointSections
		}
		if known < c.knownSections {
			c.knownSections = known
		}
		// 将存储在数据库中的部分回滚到重新组织点。
		if stored < c.storedSections {
			c.setValidSections(stored)
		}
		// 将新的头部编号更新为最终部分的结束，并通知子节点。
		head = known * c.sectionSize

		if head < c.cascadedHead {
			c.cascadedHead = head
			for _, child := range c.children {
				child.newHead(c.cascadedHead, true)
			}
		}
		return
	}
	// 不进行重组，计算新发现的部分数量，如果数量足够高则进行更新。
	var sections uint64
	if head >= c.confirmsReq {
		sections = (head + 1 - c.confirmsReq) / c.sectionSize
		if sections < c.checkpointSections {
			sections = 0
		}
		if sections > c.knownSections {
			if c.knownSections < c.checkpointSections {
				// 同步已达到检查点，请验证部分头。
				syncedHead := rawdb.ReadCanonicalHash(c.chainDb, c.checkpointSections*c.sectionSize-1)
				if syncedHead != c.checkpointHead {
					c.log.Error("Synced chain does not match checkpoint", "number", c.checkpointSections*c.sectionSize-1, "expected", c.checkpointHead, "synced", syncedHead)
					return
				}
			}
			c.knownSections = sections

			select {
			case c.update <- struct{}{}:
			default:
			}
		}
	}
}
```

父子索引数据的关系 父Indexer负载事件的监听然后把结果通过newHead传递给子Indexer的updateLoop来处理。

<img src="../../img/chainindexer_1.png">

setValidSections方法，写入当前已经存储的sections的数量。 如果传入的值小于已经存储的数量，那么从数据库里面删除对应的section
```go
// setValidSections将有效部分的数量写入索引数据库。
func (c *ChainIndexer) setValidSections(sections uint64) {
	// 在数据库中设置当前有效部分的数量。
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], sections)
	c.indexDb.Put([]byte("count"), data[:])

	// 在此期间删除任何重组的部分，并将有效部分缓存起来。
	for c.storedSections > sections {
		c.storedSections--
		c.removeSectionHead(c.storedSections)
	}
	c.storedSections = sections // needed if new > old
}
```

processSection（updateLoop中调用）
```go
// processSection通过调用后端函数来处理整个部分，同时确保传递的头部的连续性。
// 由于在处理过程中未持有链锁，长时间的重组可能会打破连续性，此时该函数将返回错误。
func (c *ChainIndexer) processSection(section uint64, lastHead common.Hash) (common.Hash, error) {
	c.log.Trace("Processing new chain section", "section", section)

	// 重置和部分处理。
	if err := c.backend.Reset(c.ctx, section, lastHead); err != nil {
		c.setValidSections(0)
		return common.Hash{}, err
	}

	for number := section * c.sectionSize; number < (section+1)*c.sectionSize; number++ {
		hash := rawdb.ReadCanonicalHash(c.chainDb, number)
		if hash == (common.Hash{}) {
			return common.Hash{}, fmt.Errorf("canonical block #%d unknown", number)
		}
		header := rawdb.ReadHeader(c.chainDb, hash, number)
		if header == nil {
			return common.Hash{}, fmt.Errorf("block #%d [%x..] not found", number, hash[:4])
		} else if header.ParentHash != lastHead {
			return common.Hash{}, errors.New("chain reorged during section processing")
		}
		if err := c.backend.Process(c.ctx, header); err != nil {
			return common.Hash{}, err
		}
		lastHead = header.Hash()
	}
	if err := c.backend.Commit(); err != nil {
		return common.Hash{}, err
	}
	return lastHead, nil
}
```








































