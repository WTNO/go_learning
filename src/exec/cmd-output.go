package main

import (
	"fmt"
	"os/exec"
)

func main() {
	c := exec.Command("/bin/bash", "-c", "vmstat 1 5")
	fmt.Printf("our==> %s", "000\n")

	// _, err := c.Output()
	// if err != nil {
	// 	fmt.Println("err==> ", err)
	// 	return
	// }

	fmt.Printf("our==> %s", "111\n")
	c.Start()
	fmt.Printf("our==> %s", "222\n")
	c.Wait()
	fmt.Printf("our==> %s", "333\n")
}
