package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"io"
)

func main() {
	fmt.Println("-----------------------------普通方案--------------------------------")
	//import "crypto/sha256"
	h1 := sha256.New()
	io.WriteString(h1, "His money is twice tainted: 'taint yours and 'taint mine.")
	fmt.Printf("% x", h1.Sum(nil))

	//import "crypto/sha1"
	h2 := sha1.New()
	io.WriteString(h2, "His money is twice tainted: 'taint yours and 'taint mine.")
	fmt.Printf("% x", h2.Sum(nil))

	//import "crypto/md5"
	h3 := md5.New()
	io.WriteString(h3, "需要加密的密码")
	fmt.Println("%x", h3.Sum(nil))

	fmt.Println("-----------------------------进阶方案--------------------------------")

	//import "crypto/md5"
	//假设用户名abc，密码123456
	h := md5.New()
	io.WriteString(h, "123456")

	//pwmd5等于e10adc3949ba59abbe56e057f20f883e
	pwmd5 := fmt.Sprintf("%x", h.Sum(nil))
	//pwmd5 := string(h.Sum(nil)) // 为什么这里是乱码？
	fmt.Println(pwmd5)

	//指定两个 salt： salt1 = @#$%   salt2 = ^&*()
	salt1 := "@#$%"
	salt2 := "^&*()"

	//salt1+用户名+salt2+MD5拼接
	io.WriteString(h, salt1)
	io.WriteString(h, "abc")
	io.WriteString(h, salt2)
	io.WriteString(h, pwmd5)

	last := fmt.Sprintf("%x", h.Sum(nil))
	fmt.Println(last)
}
