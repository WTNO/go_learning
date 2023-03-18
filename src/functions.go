package main

import "fmt"

// func <函数名>(<参数名 参数类型>) <返回类型>
func add(x int, y int) int {
	return x + y
}

// 当连续两个或多个函数的已命名形参类型相同时，除最后一个类型以外，其它都可以省略。
func plus(x, y int) int {
	return x + y
}

func main() {
	fmt.Println(add(42, 13))
}
