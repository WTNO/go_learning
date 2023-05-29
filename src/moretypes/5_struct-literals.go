package main

import "fmt"

type Vertex4 struct {
	X, Y int
}

var (
	v1 = Vertex4{1, 2}
	v2 = Vertex4{X: 1}  // Y:0 被隐式地赋予
	v3 = Vertex4{}      // X:0 Y:0
	p  = &Vertex4{1, 2} // 创建一个 *Vertex 类型的结构体（指针）
)

func main() {
	fmt.Println(v1, v2, v3, *p)
}
