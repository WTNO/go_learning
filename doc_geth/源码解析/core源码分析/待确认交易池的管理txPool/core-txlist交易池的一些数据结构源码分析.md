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






















