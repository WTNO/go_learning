package main

import (
	"fmt"
	"os"
	"regexp"
)

func IsIP(ip string) (m bool) {
	m, _ = regexp.MatchString("^[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}$", ip)
	return
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Usage: regexp [string]")
		os.Exit(1)
	}

	if m, _ := regexp.MatchString("^[0-9]+$", os.Args[1]); m {
		fmt.Println("数字")
	} else if IsIP(os.Args[1]) {
		fmt.Println("是IP")
	}

	fmt.Println(os.Args[0])
	fmt.Println(os.Args[1])
}
