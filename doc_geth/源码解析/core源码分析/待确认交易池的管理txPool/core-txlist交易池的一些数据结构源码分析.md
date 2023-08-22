## nonceHeap
nonceHeap实现了一个heap.Interface的数据结构，用来实现了一个堆的数据结构。 在heap.Interface的文档介绍中，默认实现的是最小堆。

如果h是一个数组，只要数组中的数据满足下面的要求。那么就认为h是一个最小堆。
```go
// !h.Less(j, i) for 0 <= i < h.Len() and 2*i+1 <= j <= 2*i+2 and j < h.Len()
// 
// 把数组看成是一颗满的二叉树，第一个元素是树根，第二和第三个元素是树根的两个树枝，
// 这样依次推下去 那么如果树根是 i 那么它的两个树枝就是 2*i+1 和 2*i + 2。
// 
// 最小堆的定义是 任意的树根不能比它的两个树枝大。 也就是上面的代码描述的定义。
// 
// heap.Interface的定义
// 我们只需要定义满足下面接口的数据结构，就能够使用heap的一些方法来实现为堆结构。
type Interface interface {
	sort.Interface
	Push(x interface{}) // add x as element Len() 把x增加到最后
	Pop() interface{}   //  remove and return element Len() - 1. 移除并返回最后的一个元素
}
```

### nonceHeap的代码分析
```go
// nonceHeap是一个对64位无符号整数实现的heap.Interface接口，
// 用于从可能存在间隔的未来队列中检索排序的交易。
type nonceHeap []uint64

func (h nonceHeap) Len() int           { return len(h) }
func (h nonceHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h nonceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *nonceHeap) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}

func (h *nonceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = 0
	*h = old[0 : n-1]
	return x
}
```

## txSortedMap
txSortedMap存储的是同一个账号下面的所有的交易。

### 结构
```go
// sortedMap是一个基于堆的索引的nonce->交易哈希映射，允许按照递增的nonce迭代内容。
type sortedMap struct {
	items map[uint64]*types.Transaction // 哈希映射，存储交易数据
	index *nonceHeap                    // 存储所有已存储交易的nonce的堆（非严格模式）
	cache types.Transactions            // 已排序的交易的缓存
}
```

### Put 和 Get
Get用于获取指定nonce的交易， Put用来把交易插入到map中。
```go
// Get检索与给定nonce相关联的当前交易。
func (m *sortedMap) Get(nonce uint64) *types.Transaction {
	return m.items[nonce]
}

// Put将新的交易插入映射中，同时更新映射的nonce索引。如果已存在具有相同nonce的交易，则会被覆盖。
func (m *sortedMap) Put(tx *types.Transaction) {
	nonce := tx.Nonce()
	if m.items[nonce] == nil {
		heap.Push(m.index, nonce)
	}
	m.items[nonce], m.cache = tx, nil
}
```

### Forward
用于删除所有nonce小于threshold的交易。 然后返回所有被移除的交易。
```go
// Forward函数会移除所有nonce小于提供的阈值的map中的交易。
// 每个被移除的交易都会在移除后返回以进行后续维护。
func (m *sortedMap) Forward(threshold uint64) types.Transactions {
	var removed types.Transactions

	// 弹出堆中的项，直到达到阈值
	for m.index.Len() > 0 && (*m.index)[0] < threshold {
		nonce := heap.Pop(m.index).(uint64)
		removed = append(removed, m.items[nonce])
		delete(m.items, nonce)
	}
	// 如果有缓存的顺序，将其前移
	if m.cache != nil {
		m.cache = m.cache[len(removed):]
	}
	return removed
}
```

### Filter
删除所有令filter函数调用返回true的交易，并返回那些交易。
```go
// Filter函数遍历交易列表，删除所有满足指定函数为true的交易。
// 与'filter'不同的是，Filter在操作完成后重新初始化堆。
// 如果您想进行连续的多次过滤，最好先使用.filter(func1)，然后再使用.Filter(func2)或reheap()。
func (m *sortedMap) Filter(filter func(*types.Transaction) bool) types.Transactions {
	removed := m.filter(filter)
	// 如果有交易被删除，堆和缓存将被破坏
	if len(removed) > 0 {
		m.reheap()
	}
	return removed
}

func (m *sortedMap) reheap() {
	*m.index = make([]uint64, 0, len(m.items))
	for nonce := range m.items {
		*m.index = append(*m.index, nonce)
	}
	heap.Init(m.index)
	m.cache = nil
}
```

### Cap
对items里面的数量有限制，返回超过限制的所有交易。
```go
// Cap函数对项目数量设置了一个硬限制，返回超过该限制的所有交易。
func (m *sortedMap) Cap(threshold int) types.Transactions {
	// 如果项目数量低于限制，则直接返回
	if len(m.items) <= threshold {
		return nil
	}
	// 否则收集并删除最高nonce的交易
	var drops types.Transactions

	sort.Sort(*m.index)
	for size := len(m.items); size > threshold; size-- {
		drops = append(drops, m.items[(*m.index)[size-1]])
		delete(m.items, (*m.index)[size-1])
	}
	*m.index = (*m.index)[:threshold]
	heap.Init(m.index)

	// 如果存在缓存，则将其向后移动
	if m.cache != nil {
		m.cache = m.cache[:len(m.cache)-len(drops)]
	}
	return drops
}
```

### Remove
```go
// Remove函数从维护的映射中删除一个交易，返回是否找到该交易。
func (m *sortedMap) Remove(nonce uint64) bool {
	// 如果没有交易存在，则直接返回
	_, ok := m.items[nonce]
	if !ok {
		return false
	}
	// 否则删除交易并修复堆索引
	for i := 0; i < m.index.Len(); i++ {
		if (*m.index)[i] == nonce {
			heap.Remove(m.index, i)
			break
		}
	}
	delete(m.items, nonce)
	m.cache = nil

	return true
}
```

### Ready
```go
// Ready函数检索从提供的nonce开始的顺序递增的交易列表，这些交易已准备好进行处理。
// 返回的交易将从列表中删除。
// 
// 注意，为了防止进入无效状态，还将返回所有nonce低于start的交易。
// 虽然这不应该发生，但更好的是自我纠正而不是失败！
func (m *sortedMap) Ready(start uint64) types.Transactions {
	// 如果没有可用的交易，则直接返回
	if m.index.Len() == 0 || (*m.index)[0] > start {
		return nil
	}
	// 否则开始累积增量交易
	var ready types.Transactions
	for next := (*m.index)[0]; m.index.Len() > 0 && (*m.index)[0] == next; next++ {
		ready = append(ready, m.items[next])
		delete(m.items, next)
		heap.Pop(m.index)
	}
	m.cache = nil

	return ready
}
```

### Flatten
返回一个基于nonce排序的交易列表。并缓存到cache字段里面，以便在没有修改的情况下反复使用。
```go
// Flatten函数基于松散排序的内部表示创建一个按照nonce排序的交易切片。为了防止在内容被修改之前再次请求排序结果，排序结果会被缓存起来。
func (m *sortedMap) Flatten() types.Transactions {
	// 复制缓存以防止意外修改
	cache := m.flatten()
	txs := make(types.Transactions, len(cache))
	copy(txs, cache)
	return txs
}
```

## list
list 是属于同一个账号的交易列表， 按照nonce排序。可以用来存储连续的可执行的交易。对于非连续的交易,有一些小的不同的行为。

### 结构
```go
// list 是一个按账户 nonce 排序的交易列表。
// 同一种类型可以用于存储可执行/挂起队列的连续交易；
// 也可以用于存储不可执行/未来队列的间隔交易，但会有一些行为上的差异。
type list struct {
	strict    bool       // 非ces 是否严格连续
	txs       *sortedMap // 交易的堆索引排序哈希映射

	costcap   *big.Int // 最高成本交易的价格（仅当超过余额时重置）
	gascap    uint64   // 最高消耗交易的燃气限制（仅当超过块限制时重置）
	totalcost *big.Int // 列表中所有交易的总成本
}
```

### Contains
返回给定的交易是否有具有相同nonce的交易存在。
```go
// Contains返回列表中是否包含具有给定 nonce 的交易。
func (l *list) Contains(nonce uint64) bool {
	return l.txs.Get(nonce) != nil
}
```

### Add
执行这样的操作，如果新的交易比老的交易的GasPrice值要高出一定的比值priceBump，那么会替换老的交易。
```go
// Add函数尝试将新的交易插入列表中，返回交易是否被接受，以及如果被接受，则替换的任何先前交易。
//
// 如果新的交易被接受到列表中，列表的成本和燃气阈值也可能会被更新。
func (l *list) Add(tx *types.Transaction, priceBump uint64) (bool, *types.Transaction) {
	// 如果有旧的更好的交易，则中止
	old := l.txs.Get(tx.Nonce())
	if old != nil {
		if old.GasFeeCapCmp(tx) >= 0 || old.GasTipCapCmp(tx) >= 0 {
			return false, nil
		}
		// 阈值费用上限 = oldFC * (100 + priceBump) / 100
		a := big.NewInt(100 + int64(priceBump))
		aFeeCap := new(big.Int).Mul(a, old.GasFeeCap())
		aTip := a.Mul(a, old.GasTipCap())

		// 阈值小费 = oldTip * (100 + priceBump) / 100
		b := big.NewInt(100)
		thresholdFeeCap := aFeeCap.Div(aFeeCap, b)
		thresholdTip := aTip.Div(aTip, b)

		// 我们必须确保新的费用上限和小费都高于旧的，并且检查百分比阈值以确保对于低（Wei级）燃气价格替换是准确的。
		if tx.GasFeeCapIntCmp(thresholdFeeCap) < 0 || tx.GasTipCapIntCmp(thresholdTip) < 0 {
			return false, nil
		}
		// 被替换的旧交易，减去旧的成本
		l.subTotalCost([]*types.Transaction{old})
	}
	// 将新交易成本添加到总成本中
	l.totalcost.Add(l.totalcost, tx.Cost())
	// 否则用当前交易覆盖旧交易
	l.txs.Put(tx)
	if cost := tx.Cost(); l.costcap.Cmp(cost) < 0 {
		l.costcap = cost
	}
	if gas := tx.Gas(); l.gascap < gas {
		l.gascap = gas
	}
	return true, old
}
```

### Forward
删除nonce小于某个值的所有交易。
```go
// Forward函数会删除列表中所有nonce小于提供的阈值的交易。每个被删除的交易都会在删除后返回以进行任何后续的维护。
func (l *list) Forward(threshold uint64) types.Transactions {
	txs := l.txs.Forward(threshold)
	l.subTotalCost(txs)
	return txs
}
```

### Filter
```go
// Filter函数会删除列表中所有成本或燃气限制高于提供的阈值的交易。
// 每个被删除的交易都会在删除后返回以进行任何后续的维护。严格模式下无效的交易也会被返回。
//
// 此方法使用缓存的成本限制和燃气限制来快速判断是否有必要计算所有成本，或者余额是否足够支付。
// 如果阈值低于成本和燃气限制，那么在删除新的无效交易后，限制将被重新设置为一个新的高值。
func (l *list) Filter(costLimit *big.Int, gasLimit uint64) (types.Transactions, types.Transactions) {
	// 如果所有交易都低于阈值，则立即返回
	if l.costcap.Cmp(costLimit) <= 0 && l.gascap <= gasLimit {
		return nil, nil
	}
	l.costcap = new(big.Int).Set(costLimit) // 将限制降低到阈值
	l.gascap = gasLimit

	// 过滤掉所有超过账户资金的交易
	removed := l.txs.Filter(func(tx *types.Transaction) bool {
		return tx.Gas() > gasLimit || tx.Cost().Cmp(costLimit) > 0
	})

	if len(removed) == 0 {
		return nil, nil
	}
	var invalids types.Transactions
	// 如果列表是严格模式，过滤掉任何高于最低nonce的交易
	if l.strict {
		lowest := uint64(math.MaxUint64)
		for _, tx := range removed {
			if nonce := tx.Nonce(); lowest > nonce {
				lowest = nonce
			}
		}
		invalids = l.txs.filter(func(tx *types.Transaction) bool { return tx.Nonce() > lowest })
	}
	// 重置总成本
	l.subTotalCost(removed)
	l.subTotalCost(invalids)
	l.txs.reheap()
	return removed, invalids
}
```

### Cap
用来返回超过数量的交易，如果交易的数量超过threshold,那么把之后的交易移除并返回。
```go
// Cap函数对项目数量设置了一个硬限制，返回超过该限制的所有交易。
func (l *list) Cap(threshold int) types.Transactions {
	txs := l.txs.Cap(threshold)
	l.subTotalCost(txs)
	return txs
}
```

### Remove
删除给定Nonce的交易，如果在严格模式下，还删除所有nonce大于给定Nonce的交易，并返回。
```go
// Remove函数从维护的列表中删除一个交易，返回是否找到该交易，并返回由于删除而无效的任何交易（仅在严格模式下）。
func (l *list) Remove(tx *types.Transaction) (bool, types.Transactions) {
	// 从集合中删除该交易
	nonce := tx.Nonce()
	if removed := l.txs.Remove(nonce); !removed {
		return false, nil
	}
	l.subTotalCost([]*types.Transaction{tx})
	// 在严格模式下，过滤掉不可执行的交易
	if l.strict {
		txs := l.txs.Filter(func(tx *types.Transaction) bool { return tx.Nonce() > nonce })
		l.subTotalCost(txs)
		return true, txs
	}
	return true, nil
}
```

### Ready， len, Empty, Flatten
直接调用了txSortedMap的对应方法。
```go
// Ready检索从提供的nonce开始逐步递增的交易列表，这些交易列表准备好进行处理。返回的交易将从列表中移除。
//
// 注意，所有nonce小于start的交易也将返回，以防止进入无效状态。虽然这不应该发生，但自我纠正总比失败好！
func (l *list) Ready(start uint64) types.Transactions {
	txs := l.txs.Ready(start)
	l.subTotalCost(txs)
	return txs
}

// Len返回交易列表的长度。
func (l *list) Len() int {
	return l.txs.Len()
}

// Empty返回交易列表是否为空。
func (l *list) Empty() bool {
	return l.Len() == 0
}

// Flatten根据松散排序的内部表示创建一个按nonce排序的交易切片。
// 排序结果会被缓存，以防在修改内容之前再次请求。
func (l *list) Flatten() types.Transactions {
	return l.txs.Flatten()
}
```

## priceHeap
priceHeap是一个最小堆， 按照价格的大小来建堆。
```go
// priceHeap是一个基于交易的堆接口实现，用于按价格排序的交易检索，以便在池填满时丢弃。
// 如果设置了baseFee，则堆根据给定的基础费用计算出的有效小费进行排序。
// 如果baseFee为nil，则根据gasFeeCap进行排序。
type priceHeap struct {
	baseFee *big.Int // 在baseFee更改后，堆应该重新排序
	list    []*types.Transaction
}

func (h *priceHeap) Len() int      { return len(h.list) }

func (h *priceHeap) Swap(i, j int) { h.list[i], h.list[j] = h.list[j], h.list[i] }

func (h *priceHeap) Less(i, j int) bool {
	switch h.cmp(h.list[i], h.list[j]) {
	case -1:
		return true
	case 1:
		return false
	default:
		return h.list[i].Nonce() > h.list[j].Nonce()
	}
}

func (h *priceHeap) cmp(a, b *types.Transaction) int {
	if h.baseFee != nil {
		// 如果指定了baseFee，则比较有效小费
		if c := a.EffectiveGasTipCmp(b, h.baseFee); c != 0 {
			return c
		}
	}
	// 如果未指定baseFee或有效小费相等，则比较费用上限
	if c := a.GasFeeCapCmp(b); c != 0 {
		return c
	}
	// 如果有效小费和费用上限相等，则比较小费上限
	return a.GasTipCapCmp(b)
}

func (h *priceHeap) Push(x interface{}) {
	tx := x.(*types.Transaction)
	h.list = append(h.list, tx)
}

func (h *priceHeap) Pop() interface{} {
	old := h.list
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	h.list = old[0 : n-1]
	return x
}
```

## pricedList
pricedList 是基于价格排序的堆，允许按照价格递增的方式处理交易。

### 数据结构和构建
```go
// pricedList是一个价格排序的堆，用于对交易池中的交易内容进行操作。
// 它基于txpool中的所有交易构建，但只关注远程交易部分。
// 这意味着只有远程交易会被考虑进行跟踪、排序、剔除等操作。
// 
// 使用两个堆进行排序：紧急堆（基于下一个区块的有效小费）和浮动堆（基于gasFeeCap）。
// 始终选择较大的堆进行剔除。从紧急堆中剔除的交易首先被降级到浮动堆中。
// 在某些情况下（拥堵时，区块已满），紧急堆可以提供更好的包含候选项，
// 而在其他情况下（在baseFee峰值的顶部），浮动堆更好。当baseFee降低时，它们的行为类似。

type pricedList struct {
	// 过时价格点的数量（重新堆化的触发器）。
	stales atomic.Int64

	all              *lookup    // 所有交易的映射指针
	urgent, floating priceHeap  // 存储所有**远程**交易价格的堆
	reheapMu         sync.Mutex // 互斥锁，确保只有一个例程在重新堆化列表
}

// newPricedList创建一个新的按价格排序的交易堆。
func newPricedList(all *lookup) *pricedList {
	return &pricedList{
		all: all,
	}
}
```

### Put
```go
// Put方法将一个新的交易插入堆中。
func (l *pricedList) Put(tx *types.Transaction, local bool) {
	if local {
		return
	}
	// 首先将每个新交易插入紧急堆中；Discard方法将平衡堆
	heap.Push(&l.urgent, tx)
}
```

### Removed
```go
// Removed通知价格交易列表一个旧的交易已从池中移除。
// 列表将仅保留一个过期对象的计数器，并在足够比例的交易过期时更新堆。
func (l *pricedList) Removed(count int) {
	// 增加过期计数器，但如果仍然太低（< 25%）则退出
	stales := l.stales.Add(int64(count))
	if int(stales) <= (len(l.urgent.list)+len(l.floating.list))/4 {
		return
	}
	// 看起来我们已经达到了关键数量的过期交易，重新堆化
	l.Reheap()
}
```

### ~~Cap~~

### Underpriced
检查`tx`是否比当前`pricedList`里面最便宜的交易还要便宜或者是同样便宜.
```go
// Underpriced函数检查一个交易是否比（或与）当前正在跟踪的最低价（远程）交易便宜。
func (l *pricedList) Underpriced(tx *types.Transaction) bool {
	// 注意：在有两个队列的情况下，被低估被定义为比所有非空队列中的最差项目更糟糕。
	// 如果两个队列都为空，则没有东西被低估。
	return (l.underpricedFor(&l.urgent, tx) || len(l.urgent.list) == 0) &&
		(l.underpricedFor(&l.floating, tx) || len(l.floating.list) == 0) &&
		(len(l.urgent.list) != 0 || len(l.floating.list) != 0)
}

// underpricedFor函数检查一个交易是否比给定堆中的最低价（远程）交易便宜。
func (l *pricedList) underpricedFor(h *priceHeap, tx *types.Transaction) bool {
	// 如果在堆的开头找到了过时的价格点，则丢弃它们
	for len(h.list) > 0 {
		head := h.list[0]
		if l.all.GetRemote(head.Hash()) == nil { // 被删除或迁移
			l.stales.Add(-1)
			heap.Pop(h)
			continue
		}
		break
	}
	// 检查交易是否被低估
	if len(h.list) == 0 {
		return false // 没有远程交易。
	}
	// 如果远程交易甚至比本地跟踪的最便宜的交易还便宜，则拒绝它。
	return h.cmp(h.list[0], tx) >= 0
}
```

### Discard
查找一定数量的最便宜的交易,把他们从当前的列表删除并返回
```go
// Discard 函数用于找到最低价的一些交易，并从价格列表中移除它们，然后返回这些交易以便从整个池中进一步移除。
// 如果 noPending 设置为 true，我们只会考虑浮动列表中的交易。
//
// 注意，本地交易不会被考虑在内。
func (l *pricedList) Discard(slots int, force bool) (types.Transactions, bool) {
	drop := make(types.Transactions, 0, slots) // 要移除的低价交易
	for slots > 0 {
		if len(l.urgent.list)*floatingRatio > len(l.floating.list)*urgentRatio || floatingRatio == 0 {
			// 如果在清理过程中发现陈旧的交易，则将其丢弃
			tx := heap.Pop(&l.urgent).(*types.Transaction)
			if l.all.GetRemote(tx.Hash()) == nil { // 被移除或已迁移
				l.stales.Add(-1)
				continue
			}
			// 发现非陈旧交易，移到浮动堆中
			heap.Push(&l.floating, tx)
		} else {
			if len(l.floating.list) == 0 {
				// 如果两个堆都为空，则停止
				break
			}
			// 如果在清理过程中发现陈旧的交易，则将其丢弃
			tx := heap.Pop(&l.floating).(*types.Transaction)
			if l.all.GetRemote(tx.Hash()) == nil { // 被移除或已迁移
				l.stales.Add(-1)
				continue
			}
			// 发现非陈旧交易，将其丢弃
			drop = append(drop, tx)
			slots -= numSlots(tx)
		}
	}
	// 如果仍然无法为新交易腾出足够的空间
	if slots > 0 && !force {
		for _, tx := range drop {
			heap.Push(&l.urgent, tx)
		}
		return nil, false
	}
	return drop, true
}
```












