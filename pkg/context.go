package cmdgo

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// A dynamic set of variables that commands can have access to during capture and execution.
type Context struct {
	Prompt      func(prompt string, prop Property) (string, error)
	Values      map[string]any
	HelpPrompt  string
	DisplayHelp func(prop Property)
	ArgPrefix   string
	StartIndex  int64
}

func NewContext() Context {
	reader := bufio.NewReader(os.Stdin)
	return Context{
		Values:     make(map[string]any),
		HelpPrompt: "help!",
		ArgPrefix:  "--",
		StartIndex: 1,
		Prompt: func(prompt string, prop Property) (string, error) {
			fmt.Print(prompt)
			input := ""
			for {
				line, err := reader.ReadString('\n')
				if err != nil && err != io.EOF {
					return "", err
				}
				input += line
				if !prop.PromptMulti || line == "" || err != nil {
					break
				}
			}
			input = strings.TrimRight(input, "\n")
			return input, nil
		},
		DisplayHelp: func(prop Property) {
			fmt.Println(prop.Help)
		},
	}
}
