package main

import (
	"fmt"
	"html/template"
	"os"
	"strings"
)

type Friend1 struct {
	Fname string
}

type Person1 struct {
	UserName string
	Emails   []string
	Friends  []*Friend1
}

func EmailDealWith(args ...interface{}) string {
	ok := false
	var s string
	if len(args) == 1 {
		s, ok = args[0].(string)
	}
	if !ok {
		s = fmt.Sprint(args...)
	}
	// find the @ symbol
	substrs := strings.Split(s, "@")
	if len(substrs) != 2 {
		return s
	}
	// replace the @ by " at "
	return (substrs[0] + " at " + substrs[1])
}

func main() {
	f1 := Friend1{Fname: "minux.ma"}
	f2 := Friend1{Fname: "xushiwei"}
	t := template.New("fieldname example")
	t = t.Funcs(template.FuncMap{"emailDeal": EmailDealWith})
	t, _ = t.Parse(`hello {{.UserName}}!
				{{range .Emails}}
					an emails {{.|emailDeal}}
				{{end}}
				{{with .Friends}}
				{{range .}}
					my Friend1 name is {{.Fname}}
				{{end}}
				{{end}}
				`)
	p := Person1{UserName: "Astaxie",
		Emails:  []string{"astaxie@beego.me", "astaxie@gmail.com"},
		Friends: []*Friend1{&f1, &f2}}
	t.Execute(os.Stdout, p)
}
