package cmdgo

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"

	"gopkg.in/yaml.v2"
)

var NoCommand = errors.New("No command given.")

type Registry map[string]any

type CaptureImporter func(data []byte, target any) error

func (r Registry) Add(name string, command any) {
	r[Normalize(name)] = command
}

func (r Registry) Get(name string) any {
	if command, ok := r[Normalize(name)]; ok {
		copy := cloneDefault(command)
		return copy
	}
	return nil
}

func (r Registry) Has(name string) bool {
	_, has := r[Normalize(name)]
	return has
}

func (r Registry) Execute(ctx *Context) error {
	cmd, err := r.Capture(ctx)
	if err != nil {
		return err
	}

	if executable, ok := cmd.(Executable); ok {
		return executable.Execute(ctx)
	}

	return nil
}

func (r Registry) Capture(ctx *Context) (any, error) {
	name := ""

	if len(ctx.Args) == 0 {
		if !r.Has(name) {
			return nil, NoCommand
		}
	} else {
		name = ctx.Args[0]
	}

	command := r.Get(name)

	if command == nil {
		return nil, fmt.Errorf("Command not found: %v", name)
	}

	if name != "" {
		ctx.Args = ctx.Args[1:]
	}

	interactiveDefault := "false"
	if len(ctx.Args) == 0 {
		interactiveDefault = "true"
	}

	interactive, _ := strconv.ParseBool(GetArg("interactive", interactiveDefault, &ctx.Args, ctx.ArgPrefix, true))

	for arg, importer := range CaptureImports {
		path := GetArg(arg, "", &ctx.Args, ctx.ArgPrefix, false)
		if path != "" {
			imported, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, err
			}
			err = importer(imported, command)
			if err != nil {
				return nil, err
			}
		}
	}

	if !interactive {
		ctx.DisablePrompt = true
		defer func() {
			ctx.DisablePrompt = false
		}()
	}

	commandInstance := GetInstance(command)
	err := commandInstance.Capture(ctx)

	if err != nil {
		return nil, err
	}

	return command, nil
}

var CaptureImports = map[string]CaptureImporter{
	"json": func(data []byte, target any) error {
		return json.Unmarshal(data, target)
	},
	"yaml": func(data []byte, target any) error {
		return yaml.Unmarshal(data, target)
	},
	"xml": func(data []byte, target any) error {
		return xml.Unmarshal(data, target)
	},
}
