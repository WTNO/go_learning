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


































