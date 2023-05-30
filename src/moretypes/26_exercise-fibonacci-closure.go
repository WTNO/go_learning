package main

import "fmt"

// 实现一个 fibonacci 函数，它返回一个函数（闭包），该闭包返回一个斐波纳契数列 `(0, 1, 1, 2, 3, 5, ...)`。
func fibonacci() func() int {
	a, b := 0, 1
	return func() int {
		c := a
		a, b = b, a+b
		return c
	}
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
