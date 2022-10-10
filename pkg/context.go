package cmdgo

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
)

var Quit = errors.New("QUIT")
var Discard = errors.New("DISCARD")

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
	QuitPrompt          string
	DiscardPrompt       string
	DisablePrompt       bool
	ForcePrompt         bool
	DisplayHelp         func(prop Property)
	ArgPrefix           string
	StartIndex          int
	ArgStructTemplate   *template.Template
	ArgSliceTemplate    *template.Template
	ArgArrayTemplate    *template.Template
	ArgMapKeyTemplate   *template.Template
	ArgMapValueTemplate *template.Template

	in       *os.File
	inReader *bufio.Reader
	out      *os.File
}

func (ctx *Context) WithArgs(args []string) *Context {
	ctx.Args = make([]string, len(args))
	copy(ctx.Args, args)
	return ctx
}

func (ctx *Context) ClearArgs() *Context {
	ctx.Args = []string{}
	return ctx
}

func (ctx *Context) WithFiles(in *os.File, out *os.File) *Context {
	ctx.in = in
	ctx.out = out
	ctx.inReader = bufio.NewReader(in)
	return ctx
}

func (ctx *Context) ClearFiles() *Context {
	ctx.in = nil
	ctx.out = nil
	ctx.inReader = nil
	return ctx
}

func (ctx *Context) Std() *Context {
	return ctx.WithFiles(os.Stdin, os.Stdout)
}

func (ctx *Context) Cli() *Context {
	return ctx.WithArgs(os.Args[1:])
}

func (ctx *Context) Program() *Context {
	return ctx.Std().Cli()
}

func (ctx *Context) printf(format string, args ...any) error {
	if ctx.out == nil {
		return nil
	}
	_, err := fmt.Fprintf(ctx.out, format, args...)
	return err
}

func (ctx *Context) CanPrompt() bool {
	return ctx.ForcePrompt || (ctx.in != nil && !ctx.DisablePrompt)
}

// A new context which by default has no arguments and does not support prompting.
func NewContext() *Context {
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

	var ctx *Context

	ctx = &Context{
		Args:                make([]string, 0),
		Values:              make(map[string]any),
		HelpPrompt:          "help!",
		QuitPrompt:          "quit!",
		DiscardPrompt:       "discard!",
		StartIndex:          1,
		DisablePrompt:       false,
		ForcePrompt:         false,
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
		PromptStart: func(prop Property) (bool, error) {
			if !ctx.CanPrompt() {
				return true, nil
			}
			for {
				input, err := ctx.Prompt(prop.PromptStart+ctx.PromptStartSuffix, prop)
				if err != nil {
					return false, err
				}
				if answer, ok := ctx.PromptStartOptions[strings.ToLower(input)]; ok {
					return answer, nil
				}
			}
		},
		PromptMore: func(prop Property) (bool, error) {
			if !ctx.CanPrompt() {
				return true, nil
			}
			for {
				input, err := ctx.Prompt(prop.PromptMore+ctx.PromptMoreSuffix, prop)
				if err != nil {
					return false, err
				}
				if answer, ok := ctx.PromptMoreOptions[strings.ToLower(input)]; ok {
					return answer, nil
				}
			}
		},
		PromptEnd: func(prop Property) error {
			if !ctx.CanPrompt() {
				return nil
			}
			return ctx.printf("%s\n", prop.PromptEnd)
		},
		Prompt: func(prompt string, prop Property) (string, error) {
			err := ctx.printf(prompt)
			if err != nil {
				return "", err
			}
			input := ""
			for {
				line, err := ctx.inReader.ReadString('\n')
				if err != nil && err != io.EOF {
					return "", err
				}
				input += line
				if !prop.PromptMulti || line == "" || err != nil {
					break
				}
			}
			input = strings.TrimRight(input, "\n")
			if ctx.QuitPrompt != "" && strings.EqualFold(input, ctx.QuitPrompt) {
				return input, Quit
			}
			if ctx.DiscardPrompt != "" && strings.EqualFold(input, ctx.DiscardPrompt) {
				return input, Discard
			}
			return input, nil
		},
		DisplayHelp: func(prop Property) {
			ctx.printf(prop.Help)
		},
	}

	return ctx
}

func newTemplate(pattern string) *template.Template {
	tpl, err := template.New("").Parse(pattern)
	if err != nil {
		panic(err)
	}
	return tpl
}
