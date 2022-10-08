package cmdgo

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/go-yaml/yaml"
)

func Execute(ctx *Context, args []string) error {
	cmd, err := Capture(ctx, args)
	if err != nil {
		return err
	}

	if executable, ok := cmd.(Executable); ok {
		return executable.Execute(ctx)
	}

	return nil
}

type CaptureImporter func(data []byte, target any) error

func Capture(ctx *Context, args []string) (any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("No command given.")
	}

	name := args[0]
	command := Get(name)

	if command == nil {
		return nil, fmt.Errorf("Command not found: %v", name)
	}

	args = args[1:]

	interactiveDefault := "false"
	if len(args) == 0 {
		interactiveDefault = "true"
	}

	interactive, _ := strconv.ParseBool(GetArg("interactive", interactiveDefault, &args, ctx.ArgPrefix, true))
	commandInstance := GetInstance(command)

	for arg, importer := range CaptureImports {
		path := GetArg(arg, "", &args, ctx.ArgPrefix, false)
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

	prompter := ctx.Prompt
	if prompter != nil && !interactive {
		ctx.Prompt = nil
	}

	err := commandInstance.Capture(ctx, &args)
	ctx.Prompt = prompter

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
