package main

import "fmt"

type IN2 interface {
	M()
}

type TY2 struct {
	S string
}

func (t *TY2) M() {
	if t == nil {
		fmt.Println("<nil>")
		return
	}
	fmt.Println(t.S)
}

// 即便接口内的具体值为 nil，方法仍然会被 nil 接收者调用。
// 在一些语言中，这会触发一个空指针异常，但在 Go 中通常会写一些方法来优雅地处理它（如本例中的 M 方法）。
// 注意: 保存了 nil 具体值的接口其自身并不为 nil。
func main() {
	var i IN2

	var t *TY2
	i = t
	describe2(i)
	i.M()

	i = &TY2{"hello"}
	describe2(i)
	i.M()

}

func describe2(i IN2) {
	fmt.Printf("(%v, %T)\n", i, i)
}
