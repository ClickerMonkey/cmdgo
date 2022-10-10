package main

import (
	"fmt"

	cmdgo "github.com/ClickerMonkey/cmdgo/pkg"
)

type Echo struct {
	Message string `prompt:"Enter message" help:"The message to enter" default:"Hello World" min:"2" env:"ECHO_MESSAGE" arg:"msg"`
}

var _ cmdgo.Executable = &Echo{}

func (echo *Echo) Execute(ctx *cmdgo.Context) error {
	fmt.Printf("\nECHO: %s\n", echo.Message)
	return nil
}

func main() {
	cmdgo.Register("echo", Echo{})
	cmdgo.Register("", Echo{})

	ctx := cmdgo.NewContext().Program()
	err := cmdgo.Execute(ctx)
	if err != nil {
		panic(err)
	}
}
