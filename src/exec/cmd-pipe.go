package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"
)

func main() {
	c := exec.Command("/bin/bash", "-c", "vmstat 1 5")
	fmt.Printf("out==> %s", "000\n")

	stdout, _ := c.StdoutPipe()
	stderr, _ := c.StderrPipe()

	c.Start()
	b1, _ := ioutil.ReadAll(stdout) // 过时？
	b2, _ := ioutil.ReadAll(stderr)

	fmt.Printf("1out==> %s", b1)
	fmt.Printf("2err==> %s", b2)

	c.Wait()
	b1, _ = ioutil.ReadAll(stdout)
	b2, _ = ioutil.ReadAll(stderr)

	fmt.Printf("3out==> %s", b1)
	fmt.Printf("4err==> %s", b2)
}
