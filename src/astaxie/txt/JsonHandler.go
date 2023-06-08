package main

import (
	"encoding/json"
	"fmt"
)

type Server struct {
	ServerName string
	ServerIP   string
}

type Serverslice struct {
	Servers []Server
}

var f interface{}

func main() {
	var s Serverslice
	str := `{"servers":[{"serverName":"Shanghai_VPN","serverIP":"127.0.0.1"},{"serverName":"Beijing_VPN","serverIP":"127.0.0.2"}]}`
	json.Unmarshal([]byte(str), &s)
	fmt.Println(s.Servers[0])

	b := []byte(`{"Name":"Wednesday","Age":6,"Parents":["Gomez","Morticia"]}`)
	json.Unmarshal(b, &f)
	fmt.Println(f)

	// 此时f里面存储了一个map类型，他们的key是string，值存储在空的interface{}里
	//f = map[string]interface{}{
	//	"Name": "Wednesday",
	//	"Age":  6,
	//	"Parents": []interface{}{
	//		"Gomez",
	//		"Morticia",
	//	},
	//}

	// 通过Comma-ok断言的方式访问
	m, _ := f.(map[string]interface{})

	// map的遍历是无序的
	for k, v := range m {
		switch vv := v.(type) {
		case string:
			fmt.Println(k, "is string", vv)
		case int:
			fmt.Println(k, "is int", vv)
		case float64:
			fmt.Println(k, "is float64", vv)
		case []interface{}:
			fmt.Println(k, "is an array:")
			for i, u := range vv {
				fmt.Println(i, u)
			}
		default:
			fmt.Println(k, "is of a type I don't know how to handle")
		}
	}

}
