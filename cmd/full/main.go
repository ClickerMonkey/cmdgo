package main

import "github.com/ClickerMonkey/cmdgo"

type Full struct {
	Int           int `min:"1"`
	IntDefault0   int `default:"0"`
	String        string
	StringDefault string `default:""`
}

func main() {
	cmdgo.Register(cmdgo.Entry{
		Name:    "full",
		Aliases: []string{""},
		Command: Full{},
	})

	opts := cmdgo.NewOptions().Program()
	err := cmdgo.Execute(opts)
	if err != nil {
		panic(err)
	}
}
