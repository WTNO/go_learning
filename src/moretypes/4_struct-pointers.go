package main

import "fmt"

type Vertexx struct {
	X int
	Y int
}

// 如果我们有一个指向结构体的指针 p，那么可以通过 (*p).X 来访问其字段 X。
// 不过这么写太啰嗦了，所以语言也允许我们使用隐式间接引用，直接写 p.X 就可以。
func main() {
	v := Vertexx{1, 2}
	p := &v
	p.X = 0.999e9
	fmt.Println(v)
}
