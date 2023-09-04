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

Beacon通过`CalcDifficulty`函数计算下一个区块难度，分别为不同阶段的难度创建了不同的难度计算方法，这里暂不展开描述
```go
// CalcDifficulty是难度调整算法。根据父块的时间和难度，它返回新块在给定时间创建时应具有的难度。
func (beacon *Beacon) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	// 如果尚未触发过渡，则使用传统规则进行计算
	if reached, _ := IsTTDReached(chain, parent.Hash(), parent.Number.Uint64()); !reached {
		return beacon.ethone.CalcDifficulty(chain, time, parent)
	}
	return beaconDifficulty
}
```

VerifyHeaders和VerifyHeader类似，只是VerifyHeaders进行批量校验操作。创建多个goroutine用于执行校验操作，再创建一个goroutine用于赋值控制任务分配和结果获取。最后返回一个结果channel
```go
// VerifyHeaders类似于VerifyHeader，但是可以并发地验证一批区块头。
// 该方法返回一个退出通道以中止操作，并返回一个结果通道以检索异步验证结果。
// VerifyHeaders期望区块头是有序且连续的。
func (beacon *Beacon) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	preHeaders, postHeaders, err := beacon.splitHeaders(chain, headers)
	if err != nil {
		return make(chan struct{}), errOut(len(headers), err)
	}
	if len(postHeaders) == 0 {
		return beacon.ethone.VerifyHeaders(chain, headers)
	}
	if len(preHeaders) == 0 {
		return beacon.verifyHeaders(chain, headers, nil)
	}
	// 过渡点存在于中间，将区块头分成两批，并为它们应用不同的验证规则。
	var (
		abort   = make(chan struct{})
		results = make(chan error, len(headers))
	)
	go func() {
		var (
			old, new, out      = 0, len(preHeaders), 0
			errors             = make([]error, len(headers))
			done               = make([]bool, len(headers))
			oldDone, oldResult = beacon.ethone.VerifyHeaders(chain, preHeaders)
			newDone, newResult = beacon.verifyHeaders(chain, postHeaders, preHeaders[len(preHeaders)-1])
		)
		// 收集结果
		for {
			for ; done[out]; out++ {
				results <- errors[out]
				if out == len(headers)-1 {
					return
				}
			}
			select {
			case err := <-oldResult:
				if !done[old] { // 跳过已经验证失败的TTD
					errors[old], done[old] = err, true
				}
				old++
			case err := <-newResult:
				errors[new], done[new] = err, true
				new++
			case <-abort:
				close(oldDone)
				close(newDone)
				return
			}
		}
	}()
	return abort, results
}

// verifyHeaders类似于verifyHeader，但是可以并发地验证一批头部。
// 该方法返回一个用于中止操作的quit通道和一个用于检索异步验证结果的结果通道。
// 如果相关的头部尚未在数据库中，将传递一个额外的父头部。
func (beacon *Beacon) verifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, ancestor *types.Header) (chan<- struct{}, <-chan error) {
	var (
		abort   = make(chan struct{})
		results = make(chan error, len(headers))
	)
	go func() {
		for i, header := range headers {
			var parent *types.Header
			if i == 0 {
				if ancestor != nil {
					parent = ancestor
				} else {
					parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
				}
			} else if headers[i-1].Hash() == headers[i].ParentHash {
				parent = headers[i-1]
			}
			if parent == nil {
				select {
				case <-abort:
					return
				case results <- consensus.ErrUnknownAncestor:
				}
				continue
			}
			err := beacon.verifyHeader(chain, header, parent)
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}
```

`VerifyUncles`用于叔块的校验。和校验区块头类似，叔块校验在ModeFullFake模式下直接返回校验成功。获取所有的叔块，然后遍历校验，校验失败即终止，或者校验完成返回。
```go
// VerifyUncles函数用于验证给定区块的叔块是否符合以太坊共识引擎的共识规则。
func (beacon *Beacon) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if !beacon.IsPoSHeader(block.Header()) {
		return beacon.ethone.VerifyUncles(chain, block)
	}
	// 验证是否存在叔块。在Beacon中明确禁用了叔块。
	if len(block.Uncles()) > 0 {
		return errTooManyUncles
	}
	return nil
}
```

Prepare实现共识引擎的Prepare接口，用于填充区块头的难度字段，使之符合ethash协议。这个改变是在线的。
```go
// Prepare函数实现了consensus.Engine接口，用于初始化头部的difficulty字段以符合beacon协议。
// 更改是在此函数内进行的。
func (beacon *Beacon) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	// 还没有触发过过渡，使用传统规则进行准备。
	reached, err := IsTTDReached(chain, header.ParentHash, header.Number.Uint64()-1)
	if err != nil {
		return err
	}
	if !reached {
		return beacon.ethone.Prepare(chain, header)
	}
	header.Difficulty = beaconDifficulty
	return nil
}
```

`Finalize`实现共识引擎的Finalize接口,奖励挖到区块账户和叔块账户，并填充状态树的根的值。并返回新的区块。
```go
// Finalize实现了consensus.Engine接口，并在其上处理提现交易。
func (beacon *Beacon) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, withdrawals []*types.Withdrawal) {
	if !beacon.IsPoSHeader(header) {
		beacon.ethone.Finalize(chain, header, state, txs, uncles, nil)
		return
	}
	// 提现交易处理。
	for _, w := range withdrawals {
		// 将金额从gwei转换为wei。
		amount := new(big.Int).SetUint64(w.Amount)
		amount = amount.Mul(amount, big.NewInt(params.GWei))
		state.AddBalance(w.Address, amount)
	}
	// 没有由共识层发行的区块奖励。
}
```

## Seal函数实现分析
在CPU挖矿部分，worker的taskLoop函数，执行挖矿操作的时候调用了Seal函数。

~~Seal函数尝试找出一个满足区块难度的nonce值。 在ModeFake和ModeFullFake模式下，快速返回，并且直接将nonce值取0。 在shared PoW模式下，使用shared的Seal函数。 开启threads个goroutine进行挖矿(查找符合条件的nonce值)。~~
```go
// Seal 生成给定输入块的新密封请求，并将结果推送到给定的通道中。
//
// 注意，该方法立即返回，并将以异步方式发送结果。根据共识算法，可能会返回多个结果。
func (beacon *Beacon) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	if !beacon.IsPoSHeader(block.Header()) {
		return beacon.ethone.Seal(chain, block, results, stop)
	}
	// 密封验证由外部共识引擎完成，
	// 直接返回而不推送任何块。换句话说，
	// beacon 不会通过 `results` 通道返回任何结果，这可能会永远阻塞接收方的逻辑。
	return nil
}
```




