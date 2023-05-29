package main

import "fmt"

type Vertex19 struct {
	Lat, Long float64
}

var m map[string]Vertex19 // 这里是声明

// 映射将键映射到值。
// 映射的零值为 nil 。nil 映射既没有键，也不能添加键。
// make 函数会返回给定类型的映射，并将其初始化备用。
func main() {
	m = make(map[string]Vertex19) // 这里是初始化
	m["Bell"] = Vertex19{
		23.2313, -76.3132,
	}
	fmt.Println(m["Bell"])

}
