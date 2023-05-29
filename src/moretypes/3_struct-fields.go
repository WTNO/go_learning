package main

import "fmt"

type _Vertex struct {
	X int
	Y int
}

func main() {
	v := _Vertex{1, 2}
	v.X = 4
	fmt.Println(v.X)
}
