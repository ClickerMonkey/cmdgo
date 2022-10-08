package main

import (
	"fmt"
	"os"

	cmdgo "github.com/ClickerMonkey/cmdgo/pkg"
)

type Echo struct {
	Message string `prompt:"Enter message" help:"The message to enter" default:"Hello World" min:"2" env:"ECHO_MESSAGE" arg:"msg"`
}

var _ cmdgo.Executable = &Echo{}

func (echo *Echo) Execute(ctx *cmdgo.Context) error {
	fmt.Printf("ECHO: %s\n", echo.Message)
	return nil
}

func main() {
	cmdgo.Register("echo", Echo{})

	ctx := cmdgo.NewContext(os.Args[1:])
	err := cmdgo.Execute(ctx)
	if err != nil {
		panic(err)
	}
}
