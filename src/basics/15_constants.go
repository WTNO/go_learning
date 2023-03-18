package main

import "fmt"

const Pi = 3.14

// 常量的声明与变量类似，只不过是使用 const 关键字。
// 常量不能用 := 语法声明。
func main() {
	const World = "世界"
	fmt.Println("Hello", World)
	fmt.Println("Happy", Pi, "Day")

	const Truth = true
	fmt.Println("Go rules?", Truth)
}
