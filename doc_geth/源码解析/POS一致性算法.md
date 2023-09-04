## 共识引擎描述
在挖矿部分，taskLoop执行挖矿操作的时候调用了`w.engine.Seal`函数。这里的engine是就是共识引擎。Seal为其中很重要的一个接口。它实现了nonce值的寻找和hash的计算。并且该函数是保证共识并且不能伪造的一个重要的函数。
再PoS共识算法中，Seal函数实现了权益证明。该部分源码在consensus/beacon。

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

## Beacon 实现分析
### Beacon 结构体
```go
// Beacon是一个将eth1共识和权益证明算法结合起来的共识引擎。
// 其中有一个特殊的标志来决定是使用传统的共识规则还是新规则。过渡规则在eth1/2合并规范中有描述。
// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-3675.md
//
// 这里的beacon是一个半功能的共识引擎，具有部分功能，仅用于必要的共识检查。
// 传统的共识引擎可以是实现共识接口的任何引擎（除了beacon本身）。
type Beacon struct {
	ethone consensus.Engine // 在eth1中使用的原始共识引擎，例如ethash或clique
}
```

Ethash是实现PoW的具体实现

Author方法获取了挖出这个块的矿工地址。
```go
// Author实现了consensus.Engine接口，返回块的已验证作者。
func (beacon *Beacon) Author(header *types.Header) (common.Address, error) {
	if !beacon.IsPoSHeader(header) {
		return beacon.ethone.Author(header)
	}
	return header.Coinbase, nil
}
```

VerifyHeader 用于校验区块头部信息是否符合ethash共识引擎规则。
```go
// VerifyHeader 检查一个头部是否符合股票以太坊共识引擎的共识规则。
func (beacon *Beacon) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	reached, err := IsTTDReached(chain, header.ParentHash, header.Number.Uint64()-1)
	if err != nil {
		return err
	}
	if !reached {
		return beacon.ethone.VerifyHeader(chain, header)
	}
	// 如果父区块未知，则短路处理
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// 合理性检查通过，进行正确的验证
	return beacon.verifyHeader(chain, header, parent)
}
```

然后再看看verifyHeader的实现,
```go
// verifyHeader检查一个头部是否符合以太坊共识引擎的共识规则。beacon和classic之间的区别是：
// （a）以下字段被期望为常量：
//   - difficulty被期望为0
//   - nonce被期望为0
//   - unclehash被期望为Hash(emptyHeader)
//     作为期望的常量
//
// （b）我们不再验证区块是否在未来
// （c）extradata被限制为32字节
func (beacon *Beacon) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header) error {
	// 确保头部的额外数据部分大小合理
	if len(header.Extra) > 32 {
		return fmt.Errorf("额外数据超过32字节 (%d)", len(header.Extra))
	}
	// 验证密封部分。确保nonce和uncle哈希值是期望的值。
	if header.Nonce != beaconNonce {
		return errInvalidNonce
	}
	if header.UncleHash != types.EmptyUncleHash {
		return errInvalidUncleHash
	}
	// 验证时间戳
	if header.Time <= parent.Time {
		return errInvalidTimestamp
	}
	// 验证区块的难度以确保它是默认的常量
	if beaconDifficulty.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("无效的难度: 当前为 %v, 期望为 %v", header.Difficulty, beaconDifficulty)
	}
	// 验证gas限制是否小于等于2^63-1
	if header.GasLimit > params.MaxGasLimit {
		return fmt.Errorf("无效的gasLimit: 当前为 %v, 最大值为 %v", header.GasLimit, params.MaxGasLimit)
	}
	// 验证gas使用量是否小于等于gas限制
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("无效的gasUsed: 当前为 %d, gas限制为 %d", header.GasUsed, header.GasLimit)
	}
	// 验证区块编号是否为父区块编号加1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(common.Big1) != 0 {
		return consensus.ErrInvalidNumber
	}
	// 验证头部的EIP-1559属性
	if err := misc.VerifyEIP1559Header(chain.Config(), parent, header); err != nil {
		return err
	}
	// 验证withdrawalsHash的存在/不存在
	shanghai := chain.Config().IsShanghai(header.Number, header.Time)
	if shanghai && header.WithdrawalsHash == nil {
		return errors.New("缺少withdrawalsHash")
	}
	if !shanghai && header.WithdrawalsHash != nil {
		return fmt.Errorf("无效的withdrawalsHash: 当前为 %x, 期望为nil", header.WithdrawalsHash)
	}
	// 验证excessDataGas的存在/不存在
	cancun := chain.Config().IsCancun(header.Number, header.Time)
	if !cancun && header.ExcessDataGas != nil {
		return fmt.Errorf("无效的excessDataGas: 当前为 %d, 期望为nil", header.ExcessDataGas)
	}
	if !cancun && header.DataGasUsed != nil {
		return fmt.Errorf("无效的dataGasUsed: 当前为 %d, 期望为nil", header.DataGasUsed)
	}
	if cancun {
		if err := misc.VerifyEIP4844Header(parent, header); err != nil {
			return err
		}
	}
	return nil
}
```






















