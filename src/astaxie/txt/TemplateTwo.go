package main

import (
	"html/template"
	"os"
)

type Friend struct {
	Fname string
}

type Person2 struct {
	UserName string
	Emails   []string
	Friends  []*Friend
}

func main() {
	f1 := Friend{Fname: "minux.ma"}
	f2 := Friend{Fname: "xushiwei"}
	t := template.New("fieldname example")
	t, _ = t.Parse(`hello {{.UserName}}!
			{{range .Emails}}
				an email {{. | html}}
			{{end}}
			{{with .Friends}}
			{{range .}}
				my friend name is {{.Fname}}
			{{end}}
			{{end}}
			`)
	p := Person2{
		UserName: "Astaxie",
		Emails:   []string{"astaxie@beego.me", "astaxie@gmail.com"},
		Friends:  []*Friend{&f1, &f2},
	}
	t.Execute(os.Stdout, p)
}
