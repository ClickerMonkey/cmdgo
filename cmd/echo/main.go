package main

import (
	"github.com/ClickerMonkey/cmdgo"
)

type Echo struct {
	Message string `prompt:"Enter message" help:"The message to enter" default:"Hello World" min:"2" env:"ECHO_MESSAGE" arg:"msg"`
}

var _ cmdgo.Executable = &Echo{}

func (echo *Echo) Execute(opts *cmdgo.Options) error {
	opts.Printf("\nECHO: %s\n", echo.Message)
	return nil
}

func main() {
	cmdgo.Register(cmdgo.Entry{
		Name:    "echo",
		Aliases: []string{""},
		Command: Echo{},
	})

	opts := cmdgo.NewOptions().Program()
	err := cmdgo.Execute(opts)
	if err != nil {
		panic(err)
	}
}
