## 共识引擎描述
在挖矿部分，taskLoop执行挖矿操作的时候调用了`w.engine.Seal`函数。这里的engine是就是共识引擎。Seal为其中很重要的一个接口。它实现了nonce值的寻找和hash的计算。并且该函数是保证共识并且不能伪造的一个重要的函数。
再PoW共识算法中，Seal函数实现了工作证明。该部分源码在consensus/ethhash下。

## 共识引擎接口
```go
// consensus/consensus.go
// Engine 是一个与算法无关的共识引擎接口。
type Engine interface {
	// Author 获取铸造给定区块的以太坊账户的地址，如果共识引擎基于签名，则可能与区块头的 coinbase 不同。
	Author(header *types.Header) (common.Address, error)

	// VerifyHeader 检查一个区块头是否符合给定引擎的共识规则。
	VerifyHeader(chain ChainHeaderReader, header *types.Header) error

	// VerifyHeaders 类似于 VerifyHeader，但并行地验证一批区块头。
	// 该方法返回一个退出通道用于中止操作，以及一个结果通道用于获取异步验证结果（顺序与输入切片一致）。
	VerifyHeaders(chain ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error)

	// VerifyUncles 验证给定区块的叔块是否符合给定引擎的共识规则。
	VerifyUncles(chain ChainReader, block *types.Block) error

	// Prepare 根据特定引擎的规则初始化一个区块头的共识字段。更改将直接执行。
	Prepare(chain ChainHeaderReader, header *types.Header) error

	// Finalize 运行任何事务后状态修改（例如区块奖励或提款处理），但不组装区块。
	// 注意：状态数据库可能会更新以反映在最终化时发生的任何共识规则（例如区块奖励）。
	Finalize(chain ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
		uncles []*types.Header, withdrawals []*types.Withdrawal)

	// FinalizeAndAssemble 运行任何事务后状态修改（例如区块奖励或提款处理）并组装最终区块。
	// 注意：区块头和状态数据库可能会更新以反映在最终化时发生的任何共识规则（例如区块奖励）。
	FinalizeAndAssemble(chain ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
		uncles []*types.Header, receipts []*types.Receipt, withdrawals []*types.Withdrawal) (*types.Block, error)

	// Seal 为给定输入区块生成一个新的封装请求，并将结果推送到给定通道中。
	// 注意，该方法立即返回，并将异步发送结果。根据共识算法，可能返回多个结果。
	Seal(chain ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error

	// SealHash 返回一个区块在封装之前的哈希值。
	SealHash(header *types.Header) common.Hash

	// CalcDifficulty 是难度调整算法。它返回一个新区块应该具有的难度。
	CalcDifficulty(chain ChainHeaderReader, time uint64, parent *types.Header) *big.Int

	// APIs 返回该共识引擎提供的 RPC API。
	APIs(chain ChainHeaderReader) []rpc.API

	// Close 终止共识引擎维护的任何后台线程。
	Close() error
}
```

## ethhash 实现分析
### ethhash 结构体
```go
// Ethash是基于工作量证明的共识引擎，实现了ethash算法。
type Ethash struct {
	fakeFail  *uint64        // 在假模式下，即使在假模式下也无法通过PoW检查的区块号
	fakeDelay *time.Duration // 在返回验证结果之前要延迟的时间
	fakeFull  bool           // 接受一切作为有效
}
```

Ethhash是实现PoW的具体实现，由于要使用到大量的数据集，所有有两个指向lru的指针。并且通过threads控制挖矿线程数。并在测试模式或fake模式下，简单快速处理，使之快速得到结果。

Athor方法获取了挖出这个块的矿工地址。






































