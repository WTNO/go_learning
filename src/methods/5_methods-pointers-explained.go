package main

import (
	"fmt"
	"math"
)

type Vertex5 struct {
	X, Y float64
}

// 现在我们要把 Abs 和 Scale 方法重写为函数。
func Abs5(v Vertex5) float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

func Scale(v *Vertex5, f float64) {
	v.X = v.X * f
	v.Y = v.Y * f
}

func main() {
	v := Vertex5{3, 4}
	Scale(&v, 10)
	fmt.Println(Abs5(v))
}
