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
	Prompt              func(prompt string, prop Property) (string, error)
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
func NewContext() Context {
	return NewContextFiles(os.Stdin, os.Stdout)
}

// A new context which uses the given files for prompting
func NewContextFiles(in *os.File, out *os.File) Context {
	ctx := NewContextQuiet()

	reader := bufio.NewReader(os.Stdin)

	ctx.Prompt = func(prompt string, prop Property) (string, error) {
		fmt.Fprintf(out, prompt)
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
func NewContextQuiet() Context {
	return Context{
		Values:              make(map[string]any),
		HelpPrompt:          "help!",
		StartIndex:          1,
		ArgPrefix:           "--",
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
