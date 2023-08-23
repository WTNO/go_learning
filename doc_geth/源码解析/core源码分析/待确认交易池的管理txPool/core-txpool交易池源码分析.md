txpool主要用来存放当前提交的等待写入区块的交易，有远端和本地的。

txpool里面的交易分为两种，
1. 提交但是还不能执行的，放在queue里面等待能够执行(比如说nonce太高)。
2. 等待执行的，放在pending里面等待执行。

从txpool的测试案例来看，txpool主要功能有下面几点。
1. 交易验证的功能，包括余额不足，Gas不足，Nonce太低, value值是合法的，不能为负数。
2. 能够缓存Nonce比当前本地账号状态高的交易。 存放在queue字段。 如果是能够执行的交易存放在pending字段
3. 相同用户的相同Nonce的交易只会保留一个GasPrice最大的那个。 其他的插入不成功。
4. 如果账号没有钱了，那么queue和pending中对应账号的交易会被删除。
5. 如果账号的余额小于一些交易的额度，那么对应的交易会被删除，同时有效的交易会从pending移动到queue里面。防止被广播。
6. txPool支持一些限制PriceLimit(remove的最低GasPrice限制)，PriceBump(替换相同Nonce的交易的价格的百分比) AccountSlots(每个账户的pending的槽位的最小值) GlobalSlots(全局pending队列的最大值)AccountQueue(每个账户的queueing的槽位的最小值) GlobalQueue(全局queueing的最大值) Lifetime(在queue队列的最长等待时间)
7. 有限的资源情况下按照GasPrice的优先级进行替换。
8. 本地的交易会使用journal的功能存放在磁盘上，重启之后会重新导入。 远程的交易不会。

### 数据结构
```go
// TxPool是各种特定交易池的聚合器，共同跟踪节点认为有趣的所有交易。
// 当从网络接收到交易或本地提交时，交易进入池中。
// 当交易包含在区块链中或由于资源限制而被驱逐时，它们将离开池。
type TxPool struct {
    subpools []SubPool               // 专用交易处理的子池列表
    subs     event.SubscriptionScope // 订阅范围，在关闭时取消订阅所有事件
    quit     chan chan error         // 用于关闭头部更新器的退出通道
}

// SubPool表示一个独立的专用交易池（例如，blob池）。由于无论有多少个专用池，它们都需要同步更新并组装成一个一致的视图以进行区块生产，因此此接口定义了允许主交易池管理子池的公共方法。
type SubPool interface {
	// Filter是一个选择器，用于决定是否将交易添加到此特定的子池中。
	Filter(tx *types.Transaction) bool

	// Init设置子池的基本参数，允许从磁盘加载任何保存的交易，并允许内部维护例程启动。
	//
	// 这些不应作为构造函数参数传递，也不应自行启动池，以便将多个子池保持同步。
	Init(gasTip *big.Int, head *types.Header) error

	// Close终止任何后台处理线程并释放任何持有的资源。
	Close() error

	// Reset检索区块链的当前状态，并确保交易池的内容相对于链状态是有效的。
	Reset(oldHead, newHead *types.Header)

	// SetGasTip更新子池对于新交易所需的最低价格，并删除低于此阈值的所有交易。
	SetGasTip(tip *big.Int)

	// Has返回指示子池是否具有具有给定哈希的缓存交易的指示器。
	Has(hash common.Hash) bool

	// Get如果交易包含在池中，则返回交易，否则返回nil。
	Get(hash common.Hash) *Transaction

	// Add将一批交易加入到池中，如果它们有效。由于交易频繁变动，add可能会推迟完全集成tx到稍后的时间点以批量处理多个交易。
	Add(txs []*Transaction, local bool, sync bool) []error

	// Pending检索所有当前可处理的交易，按原始账户分组并按nonce排序。
	Pending(enforceTips bool) map[common.Address][]*types.Transaction

	// SubscribeTransactions订阅新的交易事件。
	SubscribeTransactions(ch chan<- core.NewTxsEvent) event.Subscription

	// Nonce返回一个账户的下一个nonce，其中池中所有可执行的交易已经应用在其之上。
	Nonce(addr common.Address) uint64

	// Stats检索当前池的统计信息，即待处理和排队（不可执行）交易的数量。
	Stats() (int, int)

	// Content检索交易池的数据内容，返回所有待处理和排队的交易，按账户分组并按nonce排序。
	Content() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction)

	// ContentFrom检索交易池的数据内容，返回此地址的待处理和排队交易，按nonce分组。
	ContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction)

	// Locals检索当前由池认为是本地的账户。
	Locals() []common.Address

	// Status返回由哈希标识的交易的已知状态（未知/待处理/排队）。
	Status(hash common.Hash) TxStatus
}

// SubscriptionScope 提供了一种同时取消多个订阅的功能。
//
// 对于处理多个订阅的代码，可以使用作用域来方便地一次性取消所有订阅。示例演示了在较大程序中的典型用法。
//
// 零值可以直接使用。
type SubscriptionScope struct {
	mu     sync.Mutex
	subs   map[*scopeSub]struct{}
	closed bool
}
```

### 构造方法
```go
// New 创建一个新的交易池，用于从网络中收集、排序和过滤传入的交易。
func New(gasTip *big.Int, chain BlockChain, subpools []SubPool) (*TxPool, error) {
	// 检索当前头部，以便所有子池和此主协调池在初始化期间具有相同的起始状态，即使链在向前移动。
	head := chain.CurrentBlock()

	pool := &TxPool{
		subpools: subpools,
		quit:     make(chan chan error),
	}
	for i, subpool := range subpools {
		if err := subpool.Init(gasTip, head); err != nil {
			for j := i - 1; j >= 0; j-- {
				subpools[j].Close()
			}
			return nil, err
		}
	}
	go pool.loop(head, chain)
	return pool, nil
}
```

### reset(从`TxPool`移到了`SubPool`接口中，而`LegacyPool`实现了这个接口)
检索区块链的当前状态并且确保事务池的内容关于当前的区块链状态是有效的。主要功能包括：
1. 因为更换了区块头，所以原有的区块中有一些交易因为区块头的更换而作废，这部分交易需要重新加入到txPool里面等待插入新的区块
2. 生成新的currentState和pendingState
3. 因为状态的改变。将pending中的部分交易移到queue里面
4. 因为状态的改变，将queue里面的交易移入到pending里面。

```go
// reset检索区块链的当前状态，并确保交易池的内容与链状态一致。
func (pool *LegacyPool) reset(oldHead, newHead *types.Header) {
	// 如果我们正在重组旧状态，重新注入所有被丢弃的交易
	var reinject types.Transactions

	if oldHead != nil && oldHead.Hash() != newHead.ParentHash {
		// 如果重组过深，避免执行（在快速同步期间会发生）
		oldNum := oldHead.Number.Uint64()
		newNum := newHead.Number.Uint64()

		if depth := uint64(math.Abs(float64(oldNum) - float64(newNum))); depth > 64 {
			log.Debug("跳过深度交易重组", "深度", depth)
		} else {
			// 重组似乎足够浅，将所有交易都拉入内存
			var discarded, included types.Transactions
			var (
				rem = pool.chain.GetBlock(oldHead.Hash(), oldHead.Number.Uint64())
				add = pool.chain.GetBlock(newHead.Hash(), newHead.Number.Uint64())
			)
			if rem == nil {
				// 如果执行了setHead操作，可能会发生这种情况，我们只是从链中丢弃旧的头部。
				// 如果是这种情况，我们不再拥有丢失的交易，也没有添加的内容
				if newNum >= oldNum {
					// 如果重组到了相同或更高的编号，则不是setHead的情况
					log.Warn("交易池重置，丢失了旧头部",
						"旧", oldHead.Hash(), "旧编号", oldNum, "新", newHead.Hash(), "新编号", newNum)
					return
				}
				// 如果重组结果是较低的编号，表明是setHead造成的
				log.Debug("跳过由setHead引起的交易重置",
					"旧", oldHead.Hash(), "旧编号", oldNum, "新", newHead.Hash(), "新编号", newNum)
				// 我们仍然需要更新当前状态，以便用户可以重新添加丢失的交易
			} else {
				for rem.NumberU64() > add.NumberU64() {
					discarded = append(discarded, rem.Transactions()...)
					if rem = pool.chain.GetBlock(rem.ParentHash(), rem.NumberU64()-1); rem == nil {
						log.Error("交易池看到未根的旧链", "块", oldHead.Number, "哈希", oldHead.Hash())
						return
					}
				}
				for add.NumberU64() > rem.NumberU64() {
					included = append(included, add.Transactions()...)
					if add = pool.chain.GetBlock(add.ParentHash(), add.NumberU64()-1); add == nil {
						log.Error("交易池看到未根的新链", "块", newHead.Number, "哈希", newHead.Hash())
						return
					}
				}
				for rem.Hash() != add.Hash() {
					discarded = append(discarded, rem.Transactions()...)
					if rem = pool.chain.GetBlock(rem.ParentHash(), rem.NumberU64()-1); rem == nil {
						log.Error("交易池看到未根的旧链", "块", oldHead.Number, "哈希", oldHead.Hash())
						return
					}
					included = append(included, add.Transactions()...)
					if add = pool.chain.GetBlock(add.ParentHash(), add.NumberU64()-1); add == nil {
						log.Error("交易池看到未根的新链", "块", newHead.Number, "哈希", newHead.Hash())
						return
					}
				}
				reinject = types.TxDifference(discarded, included)
			}
		}
	}
	// 将内部状态初始化为当前头部
	if newHead == nil {
		newHead = pool.chain.CurrentBlock() // 在测试期间的特殊情况
	}
	statedb, err := pool.chain.StateAt(newHead.Root)
	if err != nil {
		log.Error("重置交易池状态失败", "错误", err)
		return
	}
	pool.currentHead.Store(newHead)
	pool.currentState = statedb
	pool.pendingNonces = newNoncer(statedb)

	// 注入由于重组而丢弃的任何交易
	log.Debug("重新注入过期交易", "数量", len(reinject))
	core.SenderCacher.Recover(pool.signer, reinject)
	pool.addTxsLocked(reinject, false)
}
```
被`LegacyPool`的`Init`方法调用，而在`TxPool`的`New`方法中循环调用了`SubPool`的`Init`方法





























