package cmdgo

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// A dynamic set of variables that commands can have access to during capture and execution.
type CommandContext struct {
	Prompt      func(prompt string) (string, error)
	Values      map[string]any
	HelpPrompt  string
	DisplayHelp func(prop CommandProperty)
	ArgPrefix   string
}

func NewStandardContext(values map[string]any) CommandContext {
	reader := bufio.NewReader(os.Stdin)
	return CommandContext{
		Values:     values,
		HelpPrompt: "help!",
		ArgPrefix:  "--",
		Prompt: func(prompt string) (string, error) {
			fmt.Print(prompt)
			input, err := reader.ReadString('\n')
			if err != nil {
				return "", err
			}
			input = strings.TrimRight(input, "\n")
			return input, nil
		},
		DisplayHelp: func(prop CommandProperty) {
			fmt.Println(prop.Help)
		},
	}
}
