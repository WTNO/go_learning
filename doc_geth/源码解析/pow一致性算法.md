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
```go
// Author 实现了 consensus.Engine 接口，通过验证工作量证明来确定区块的 coinbase 地址作为区块的作者。
func (ethash *Ethash) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}
```

VerifyHeader 用于校验区块头部信息是否符合ethash共识引擎规则。
```go
// VerifyHeader 检查一个头部是否符合以太坊 ethash 引擎的共识规则。
func (ethash *Ethash) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	// 如果头部已知，或者其父块未知，则进行短路处理(直接返回)
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil { // 获取父结点失败
		return consensus.ErrUnknownAncestor
	}
	// 通过了基本的检查，进行正式的验证
	return ethash.verifyHeader(chain, header, parent, false, time.Now().Unix())
}
```

然后再看看verifyHeader的实现,
```go
// verifyHeader检查一个头部是否符合以太坊ethash引擎的共识规则。
// 参考YP第4.3.4节“区块头有效性”
func (ethash *Ethash) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header, uncle bool, unixNow int64) error {
	// 确保头部的额外数据部分长度合理
	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("额外数据过长: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}
	// 验证头部的时间戳
	if !uncle {
		if header.Time > uint64(unixNow+allowedFutureBlockTimeSeconds) {
			return consensus.ErrFutureBlock
		}
	}
	if header.Time <= parent.Time {
		return errOlderBlockTime
	}
	// 根据时间戳和父区块的难度，验证区块的难度
	expected := ethash.CalcDifficulty(chain, header.Time, parent)

	if expected.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("无效的难度: 当前 %v，期望 %v", header.Difficulty, expected)
	}
	// 验证 gas 限制是否 <= 2^63-1
	if header.GasLimit > params.MaxGasLimit {
		return fmt.Errorf("无效的 gasLimit: 当前 %v，最大值 %v", header.GasLimit, params.MaxGasLimit)
	}
	// 验证 gasUsed 是否 <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("无效的 gasUsed: 当前 %d，gasLimit %d", header.GasUsed, header.GasLimit)
	}
	// 验证区块的 gas 使用情况和（如果适用）验证基础费用。
	if !chain.Config().IsLondon(header.Number) {
		// 在 EIP-1559 软分叉之前，验证 BaseFee 不存在。
		if header.BaseFee != nil {
			return fmt.Errorf("分叉前无效的 baseFee: 当前 %d，期望为 'nil'", header.BaseFee)
		}
		if err := misc.VerifyGaslimit(parent.GasLimit, header.GasLimit); err != nil {
			return err
		}
	} else if err := misc.VerifyEIP1559Header(chain.Config(), parent, header); err != nil {
		// 验证头部的 EIP-1559 属性。
		return err
	}
	// 验证区块号是父区块号加1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}
	if chain.Config().IsShanghai(header.Number, header.Time) {
		return errors.New("ethash不支持shanghai硬分叉")
	}
	if chain.Config().IsCancun(header.Number, header.Time) {
		return errors.New("ethash不支持cancun硬分叉")
	}
	// 为测试添加一些伪检查
	if ethash.fakeDelay != nil {
		time.Sleep(*ethash.fakeDelay)
	}
	if ethash.fakeFail != nil && *ethash.fakeFail == header.Number.Uint64() {
		return errors.New("无效的测试者pow")
	}
	// 如果所有检查通过，验证硬分叉的特殊字段
	if err := misc.VerifyDAOHeaderExtraData(chain.Config(), header); err != nil {
		return err
	}
	return nil
}
```

Ethash通过CalcDifficulty函数计算下一个区块难度，分别为不同阶段的难度创建了不同的难度计算方法，这里暂不展开描述
```go
// CalcDifficulty 是难度调整算法。在给定父区块的时间和难度的情况下，
// 它返回新区块在创建时应该具有的难度。
func (ethash *Ethash) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return CalcDifficulty(chain.Config(), time, parent)
}

// CalcDifficulty 是难度调整算法。在给定父区块的时间和难度的情况下，
// 它返回新区块在创建时应该具有的难度。
func CalcDifficulty(config *params.ChainConfig, time uint64, parent *types.Header) *big.Int {
	next := new(big.Int).Add(parent.Number, big1)
	switch {
	case config.IsGrayGlacier(next):
		return calcDifficultyEip5133(time, parent)
	case config.IsArrowGlacier(next):
		return calcDifficultyEip4345(time, parent)
	case config.IsLondon(next):
		return calcDifficultyEip3554(time, parent)
	case config.IsMuirGlacier(next):
		return calcDifficultyEip2384(time, parent)
	case config.IsConstantinople(next):
		return calcDifficultyConstantinople(time, parent)
	case config.IsByzantium(next):
		return calcDifficultyByzantium(time, parent)
	case config.IsHomestead(next):
		return calcDifficultyHomestead(time, parent)
	default:
		return calcDifficultyFrontier(time, parent)
	}
}
```

VerifyHeaders和VerifyHeader类似，只是VerifyHeaders进行批量校验操作。创建多个goroutine用于执行校验操作，再创建一个goroutine用于赋值控制任务分配和结果获取。最后返回一个结果channel
```go
// VerifyHeaders类似于VerifyHeader，但可以并发地验证一组头部。
// 该方法返回一个用于中止操作的退出通道和一个用于检索异步验证结果的结果通道。
func (ethash *Ethash) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	// 如果我们正在运行一个模拟的完全引擎，接受任何输入都是有效的
	if ethash.fakeFull || len(headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}
	abort := make(chan struct{})
	results := make(chan error, len(headers))
	unixNow := time.Now().Unix()

	go func() {
		for i, header := range headers {
			var parent *types.Header
			if i == 0 {
				parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
			} else if headers[i-1].Hash() == headers[i].ParentHash {
				parent = headers[i-1]
			}
			var err error
			if parent == nil {
				err = consensus.ErrUnknownAncestor
			} else {
				err = ethash.verifyHeader(chain, header, parent, false, unixNow)
			}
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

`VerifyHeaders` 在校验单个区块头的时候使用了 `verifyHeader` 

`VerifyUncles`用于叔块的校验。和校验区块头类似，叔块校验在ModeFullFake模式下直接返回校验成功。获取所有的叔块，然后遍历校验，校验失败即终止，或者校验完成返回。
```go
// VerifyUncles函数验证给定块的叔块是否符合以太坊ethash引擎的共识规则。
func (ethash *Ethash) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	// 如果我们运行的是全功能引擎模拟，接受任何输入作为有效
	if ethash.fakeFull {
		return nil
	}
	// 验证此块中包含的叔块数量最多为2个
	if len(block.Uncles()) > maxUncles {
		return errTooManyUncles
	}
	if len(block.Uncles()) == 0 {
		return nil
	}
	// 收集过去的叔块和祖先的集合
	uncles, ancestors := mapset.NewSet[common.Hash](), make(map[common.Hash]*types.Header)

	number, parent := block.NumberU64()-1, block.ParentHash()
	for i := 0; i < 7; i++ {
		ancestorHeader := chain.GetHeader(parent, number)
		if ancestorHeader == nil {
			break
		}
		ancestors[parent] = ancestorHeader
		// 如果祖先没有任何叔块，我们不必迭代它们
		if ancestorHeader.UncleHash != types.EmptyUncleHash {
			// 还需要将这些叔块添加到禁止列表中
			ancestor := chain.GetBlock(parent, number)
			if ancestor == nil {
				break
			}
			for _, uncle := range ancestor.Uncles() {
				uncles.Add(uncle.Hash())
			}
		}
		parent, number = ancestorHeader.ParentHash, number-1
	}
	ancestors[block.Hash()] = block.Header()
	uncles.Add(block.Hash())

	// 验证每个叔块是否最近，但不是祖先
	for _, uncle := range block.Uncles() {
		// 确保每个叔块只获得一次奖励
		hash := uncle.Hash()
		if uncles.Contains(hash) {
			return errDuplicateUncle
		}
		uncles.Add(hash)

		// 确保叔块具有有效的祖先
		if ancestors[hash] != nil {
			return errUncleIsAncestor
		}
		if ancestors[uncle.ParentHash] == nil || uncle.ParentHash == block.ParentHash() {
			return errDanglingUncle
		}
		if err := ethash.verifyHeader(chain, uncle, ancestors[uncle.ParentHash], true, time.Now().Unix()); err != nil {
			return err
		}
	}
	return nil
}
```

Prepare实现共识引擎的Prepare接口，用于填充区块头的难度字段，使之符合ethash协议。这个改变是在线的。
```go
// Prepare 实现了 consensus.Engine 接口，用于初始化一个头部的 difficulty 字段，以符合 ethash 协议。更改是在代码中完成的。

func (ethash *Ethash) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor // 如果找不到父区块，则返回未知的祖先错误
	}
	header.Difficulty = ethash.CalcDifficulty(chain, header.Time, parent) // 计算并设置头部的难度字段
	return nil
}
```

`Finalize`实现共识引擎的Finalize接口,奖励挖到区块账户和叔块账户，并填充状态树的根的值。并返回新的区块。
```go
// Finalize实现了consensus.Engine接口，用于累积区块和叔块的奖励。
func (ethash *Ethash) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, withdrawals []*types.Withdrawal) {
    // 累积区块和叔块的奖励
    accumulateRewards(chain.Config(), state, header, uncles)
}

// AccumulateRewards 函数将给定区块的 coinbase 账户增加挖矿奖励。
// 总奖励包括静态的区块奖励和包含的叔叔区块的奖励。
// 每个叔叔区块的 coinbase 账户也会被奖励。
func accumulateRewards(config *params.ChainConfig, state *state.StateDB, header *types.Header, uncles []*types.Header) {
	// 根据链的进展选择正确的区块奖励
	blockReward := FrontierBlockReward
	if config.IsByzantium(header.Number) {
		blockReward = ByzantiumBlockReward
	}
	if config.IsConstantinople(header.Number) {
		blockReward = ConstantinopleBlockReward
	}
	// 累积矿工和任何包含的叔叔区块的奖励
	reward := new(big.Int).Set(blockReward)
	r := new(big.Int)
	for _, uncle := range uncles {
		r.Add(uncle.Number, big8)
		r.Sub(r, header.Number)
		r.Mul(r, blockReward)
		r.Div(r, big8)
		state.AddBalance(uncle.Coinbase, r) // 增加叔叔区块的奖励到 coinbase 账户

		r.Div(blockReward, big32)
		reward.Add(reward, r)
	}
	state.AddBalance(header.Coinbase, reward) // 增加矿工的奖励到 coinbase 账户
}
```

## Seal函数实现分析
~~在CPU挖矿部分，CpuAgent的mine函数，执行挖矿操作的时候调用了Seal函数。~~

Seal函数尝试找出一个满足区块难度的nonce值。 在ModeFake和ModeFullFake模式下，快速返回，并且直接将nonce值取0。 在shared PoW模式下，使用shared的Seal函数。 开启threads个goroutine进行挖矿(查找符合条件的nonce值)。
```go

// Seal 为给定的输入区块生成一个新的封闭请求，并将结果推送到给定的通道中。
// 对于 ethash 引擎，该方法将会触发 panic，因为不再支持封闭操作。
func (ethash *Ethash) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
    panic("ethash (pow) 不再支持封闭操作")
}
```






















