package main

import (
	"fmt"
	"time"
)

// 当 select 中的其它分支都没有准备好时，default 分支就会执行。
// 为了在尝试发送或者接收时不发生阻塞，可使用 default 分支：
// select {
// case i := <-c:
//
//	// 使用 i
//
// default:
//
//	    // 从 c 中接收会阻塞时执行
//	}
func main() {
	tick := time.Tick(100 * time.Millisecond)  // 相当于定时器每隔0.1秒
	boom := time.After(500 * time.Millisecond) // 0.5秒后

	for {
		select {
		case <-tick:
			fmt.Println("tick")
		case <-boom:
			fmt.Println("BOOM")
			return
		default:
			fmt.Println("   .")
			time.Sleep(10 * time.Millisecond)
		}
	}
}
