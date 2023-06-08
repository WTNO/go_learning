package main

import (
	"fmt"
	"os"
	"text/template"
)

func main() {
	s1, error := template.ParseFiles("header.tmpl", "content.tmpl", "footer.tmpl")
	if error != nil {
		fmt.Println(error)
	}
	s1.ExecuteTemplate(os.Stdout, "header", nil)
	fmt.Println(1)
	s1.ExecuteTemplate(os.Stdout, "content", nil)
	fmt.Println(2)
	s1.ExecuteTemplate(os.Stdout, "footer", nil)
	fmt.Println(3)
	s1.Execute(os.Stdout, nil)
}
