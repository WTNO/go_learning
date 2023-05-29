package main

import (
	"fmt"
	"runtime"
)

// Go 只运行选定的 case，而非之后所有的 case。
// 实际上，Go 自动提供了在这些语言中每个 case 后面所需的 break 语句。
// 除非以 fallthrough 语句结束，否则分支会自动终止。
// Go 的另一点重要的不同在于 switch 的 case 无需为常量，且取值不必为整数。
func main() {
	fmt.Print("Go runs on ")
	switch os := runtime.GOOS; os {
	case "darwin":
		fmt.Println("OS X.")
	case "linux":
		fmt.Println("Linux.")
	default:
		fmt.Printf("%s.\n", os)
	}
}
