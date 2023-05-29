package main

import "fmt"

// 为切片追加新的元素是种常用的操作，为此 Go 提供了内建的 append 函数。
// func append(s []T, vs ...T) []T
// append 的第一个参数 s 是一个元素类型为 T 的切片，其余类型为 T 的值将会追加到该切片的末尾。
// append 的结果是一个包含原切片所有元素加上新添加元素的切片。
// 当 s 的底层数组太小，不足以容纳所有给定的值时，它就会分配一个更大的数组。返回的切片会指向这个新分配的数组。
func main() {
	var s []int
	printSlice2(s)

	s = append(s, 0)
	printSlice2(s)

	s = append(s, 1)
	printSlice2(s)

	s = append(s, 2, 3, 4)
	printSlice2(s)
}

func printSlice2(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
