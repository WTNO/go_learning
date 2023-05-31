package main

import "fmt"

// Go 的返回值可被命名，它们会被视作定义在函数顶部的变量。
// 没有参数的 return 语句返回已命名的返回值。也就是 直接 返回。
// 直接返回语句 应当 仅用在下面这样的短函数中。在长的函数中它们会影响代码的可读性。
func split(sum int) (x, y int) {
	x = sum * 4 / 9
	y = sum - x
	return
}

func main() {
	fmt.Println(split(17))
}
