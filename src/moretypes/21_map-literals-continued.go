package main

import "fmt"

type Vertex21 struct {
	Lat, Long float64
}

var x = map[string]Vertex21{
	"Bell Labs": {40.68433, -74.39967},
	"Google":    {37.42202, -122.08408},
}

func main() {
	fmt.Println(x)
}
