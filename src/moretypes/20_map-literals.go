package main

import "fmt"

type Vertex20 struct {
	Lat, Long float64
}

var mm = map[string]Vertex20{
	"A": {40.23, 21.44},
	"B": {20.223, 11.44},
}

// 映射的文法与结构体相似，不过必须有键名。
func main() {
	fmt.Println(mm)
}
