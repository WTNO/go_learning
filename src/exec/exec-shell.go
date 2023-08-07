package main

/* Golang语言执行linux命令行 */

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"time"
)

func run() {
	cmd := exec.Command("/bin/bash", "-c", "ping 127.0.0.1")
	// 命令的输出直接扔掉
	_, err := cmd.Output()
	// 命令出错
	if err != nil {
		panic(err.Error())
	}
	// 命令启动和启动时出错
	if err := cmd.Start(); err != nil {
		panic(err.Error())
	}
	// 等待结束
	if err := cmd.Wait(); err != nil {
		panic(err.Error())
	}
}
func main() {
	// 异步线程
	go run()
	fmt.Println(time.Now().Format("2006.01.02 15:04:05"))
	// 等待1秒
	time.Sleep(1e9)
	cmd := exec.Command("/bin/bash", "-c", `ps -ef | grep -v "grep" | grep "ping"`)
	// 接收命令的标准输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("StdoutPipe: " + err.Error())
		return
	}
	// 接受命令的标准错误
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println("StderrPipe: ", err.Error())
		return
	}
	// 启动
	if err := cmd.Start(); err != nil {
		fmt.Println("Start: ", err.Error())
		return
	}
	// 读取错误信息
	bytesErr, err := ioutil.ReadAll(stderr)
	if err != nil {
		fmt.Println("ReadAll stderr: ", err.Error())
		return
	}
	if len(bytesErr) != 0 {
		fmt.Printf("stderr is not nil: %s", bytesErr)
		return
	}
	// 读取输出
	bytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		fmt.Println("ReadAll stdout: ", err.Error())
		return
	}
	// 等等命令执行完成
	if err := cmd.Wait(); err != nil {
		fmt.Println("Wait: ", err.Error())
		return
	}
	// 打印输出
	fmt.Printf("stdout: %s", bytes)
	// 等待主程序退出,携程退出.
	time.Sleep(1e9 * 10)
}
