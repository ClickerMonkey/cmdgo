package cmdgo

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// An error returned when no command to capture/execute is given in the arguments.
var NoCommand = errors.New("No command given.")

// A map of "commands" by name.
type Registry map[string]any

// An importer that can apply data to a target before Capture is executed.
type CaptureImporter func(data []byte, target any) error

// Adds a command to the registry with the given name. The name is normalized, essentially ignoring case and punctuation.
func (r Registry) Add(name string, command any) {
	r[Normalize(name)] = command
}

// Converts the partial name into a matching command name. If no
// match could be found then an empty string is returned.
func (r Registry) Convert(namePartial string) (string, bool) {
	name := Normalize(namePartial)

	if _, ok := r[name]; ok {
		return name, true
	}

	lastKey := ""
	valueCount := 0

	for key := range r {
		if strings.HasPrefix(key, name) {
			lastKey = key
			valueCount++
		}
	}

	if valueCount == 1 {
		return lastKey, true
	}

	return "", false
}

// Gets an instance of a command with the given name, or nil if non could be found.
func (r Registry) Get(name string) any {
	converted, exists := r.Convert(name)
	if !exists {
		return nil
	}
	command, _ := r[converted]
	copy := cloneDefault(command)
	return copy
}

// Returns whether the registry has a command with the given name.
func (r Registry) Has(name string) bool {
	_, has := r.Convert(name)
	return has
}

// Executes an executable command based on the given options.
func (r Registry) Execute(opts *Options) error {
	cmd, err := r.Capture(opts)
	if err != nil {
		return err
	}

	if executable, ok := cmd.(Executable); ok {
		return executable.Execute(opts)
	}

	return nil
}

// Captures a command from the options and returns it. The first argument in the options is expected to be the name of the command. If no arguments are given the default "" command is used.
// The remaining arguments are used to populate the value.
// If no arguments are specified beyond the name then interactive mode is enabled by default.
// Interactive (prompt) can be disabled entirely with "--interactive false".
// Importers are also evaluted, like --json, --xml, and --yaml. The value following is the path to the file to import.
func (r Registry) Capture(opts *Options) (any, error) {
	name := ""

	if len(opts.Args) == 0 {
		if !r.Has(name) {
			return nil, NoCommand
		}
	} else {
		name = opts.Args[0]
	}

	command := r.Get(name)

	if command == nil {
		return nil, fmt.Errorf("Command not found: %v", name)
	}

	if name != "" {
		opts.Args = opts.Args[1:]
	}

	interactiveDefault := "false"
	if len(opts.Args) == 0 {
		interactiveDefault = "true"
	}

	interactive, _ := strconv.ParseBool(GetArg("interactive", interactiveDefault, &opts.Args, opts.ArgPrefix, true))

	for arg, importer := range CaptureImports {
		path := GetArg(arg, "", &opts.Args, opts.ArgPrefix, false)
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
		opts.DisablePrompt = true
		defer func() {
			opts.DisablePrompt = false
		}()
	}

	commandInstance := GetInstance(command)
	err := commandInstance.Capture(opts)

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
