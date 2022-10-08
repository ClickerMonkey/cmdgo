package cmdgo

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
)

// A dynamic set of variables that commands can have access to during capture and execution.
type Context struct {
	Args                []string
	Prompt              func(prompt string, prop Property) (string, error)
	PromptStart         func(prop Property) (bool, error)
	PromptStartOptions  map[string]bool
	PromptStartSuffix   string
	PromptMore          func(prop Property) (bool, error)
	PromptMoreOptions   map[string]bool
	PromptMoreSuffix    string
	PromptEnd           func(prop Property) error
	Values              map[string]any
	HelpPrompt          string
	DisplayHelp         func(prop Property)
	ArgPrefix           string
	StartIndex          int
	ArgStructTemplate   *template.Template
	ArgSliceTemplate    *template.Template
	ArgArrayTemplate    *template.Template
	ArgMapKeyTemplate   *template.Template
	ArgMapValueTemplate *template.Template
}

// A new context which uses std in & out for prompting
func NewContext(args []string) *Context {
	return NewContextFiles(args, os.Stdin, os.Stdout)
}

// A new context which uses the given files for prompting
func NewContextFiles(args []string, in *os.File, out *os.File) *Context {
	ctx := NewContextQuiet(args)

	reader := bufio.NewReader(os.Stdin)

	ctx.PromptStart = func(prop Property) (bool, error) {
		for {
			input, err := ctx.Prompt(prop.PromptStart+ctx.PromptStartSuffix, prop)
			if err != nil {
				return false, err
			}
			if answer, ok := ctx.PromptStartOptions[strings.ToLower(input)]; ok {
				return answer, nil
			}
		}
	}

	ctx.PromptMore = func(prop Property) (bool, error) {
		for {
			input, err := ctx.Prompt(prop.PromptMore+ctx.PromptMoreSuffix, prop)
			if err != nil {
				return false, err
			}
			if answer, ok := ctx.PromptMoreOptions[strings.ToLower(input)]; ok {
				return answer, nil
			}
		}
	}

	ctx.PromptEnd = func(prop Property) error {
		_, err := fmt.Fprintf(out, "%s\n", prop.PromptEnd)
		return err
	}

	ctx.Prompt = func(prompt string, prop Property) (string, error) {
		_, err := fmt.Fprintf(out, prompt)
		if err != nil {
			return "", err
		}
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
	}

	ctx.DisplayHelp = func(prop Property) {
		fmt.Fprintf(out, prop.Help)
	}

	return ctx
}

// A new context which does not support prompting.
func NewContextQuiet(args []string) *Context {
	argsCopy := make([]string, len(args))
	copy(argsCopy, args)

	promptOptions := map[string]bool{
		"y":     true,
		"yes":   true,
		"1":     true,
		"t":     true,
		"true":  true,
		"ok":    true,
		"":      true,
		"n":     false,
		"no":    false,
		"0":     false,
		"f":     false,
		"false": false,
	}

	return &Context{
		Args:                argsCopy,
		Values:              make(map[string]any),
		HelpPrompt:          "help!",
		StartIndex:          1,
		ArgPrefix:           "--",
		PromptStartOptions:  promptOptions,
		PromptStartSuffix:   " (y/n): ",
		PromptMoreOptions:   promptOptions,
		PromptMoreSuffix:    " (y/n): ",
		ArgStructTemplate:   newTemplate("{{ .Prefix }}{{ .Arg }}-"),
		ArgSliceTemplate:    newTemplate("{{ .Prefix }}{{ .Arg }}{{ if not .IsSimple }}-{{ .Index }}-{{ end }}"),
		ArgArrayTemplate:    newTemplate("{{ .Prefix }}{{ .Arg }}-{{ .Index }}{{ if not .IsSimple }}-{{ end }}"),
		ArgMapKeyTemplate:   newTemplate("{{ .Prefix }}{{ .Arg }}-key{{ if not .IsSimple }}-{{ end }}"),
		ArgMapValueTemplate: newTemplate("{{ .Prefix }}{{ .Arg }}-value{{ if not .IsSimple }}-{{ end }}"),
	}
}

func newTemplate(pattern string) *template.Template {
	tpl, err := template.New("").Parse(pattern)
	if err != nil {
		panic(err)
	}
	return tpl
}
