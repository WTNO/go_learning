package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// 小写不会导出
type Server1 struct {
	ServerName string
	ServerIP   string
}

type Server2 struct {
	// ID 不会导出到JSON中
	ID int `json:"-"`

	// ServerName2 的值会进行二次JSON编码
	ServerName  string `json:"serverName"`
	ServerName2 string `json:"serverName2,string"`

	// 如果 ServerIP 为空，则不输出到JSON串中
	ServerIP string `json:"serverIP,omitempty"`
}

type Serverslice1 struct {
	Servers []Server1
}

func main() {
	var s Serverslice1
	s.Servers = append(s.Servers, Server1{ServerName: "Shanghai_VPN", ServerIP: "127.0.0.1"})
	s.Servers = append(s.Servers, Server1{ServerName: "Beijing_VPN", ServerIP: "127.0.0.2"})
	b, err := json.Marshal(s)
	if err != nil {
		fmt.Println("json err:", err)
	}
	fmt.Println(string(b))

	str := Server2{
		ID:          3,
		ServerName:  `Go "1.0" `,
		ServerName2: `Go "1.0" `,
		ServerIP:    ``,
	}
	byteArr, _ := json.Marshal(str)
	os.Stdout.Write(byteArr)
}
