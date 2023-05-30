package main

import (
	"fmt"
	"time"
)

type MyError struct {
	When time.Time
	What string
}

func (e *MyError) Error() string {
	return fmt.Sprintf("at %v, %s", e.When, e.What)
}

func run() error {
	return &MyError{
		time.Now(),
		"it didn't work",
	}
}

// Go 程序使用 error 值来表示错误状态。
//
// 与 fmt.Stringer 类似，error 类型是一个内建接口：
//
//	type error interface {
//	    Error() string
//	}
//
// （与 fmt.Stringer 类似，fmt 包在打印值时也会满足 error。）
// 通常函数会返回一个 error 值，调用的它的代码应当判断这个错误是否等于 nil 来进行错误处理。
func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	}
}
