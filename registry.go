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
var NoCommand = errors.New("No command given, try running with --help.")

// A map of "commands" by name.
type Registry struct {
	entries  []*RegistryEntry
	entryMap map[string]*RegistryEntry
}

func NewRegistry() Registry {
	return Registry{
		entries:  make([]*RegistryEntry, 0),
		entryMap: make(map[string]*RegistryEntry),
	}
}

// An entry for a registered command.
type RegistryEntry struct {
	// The user friendly name of the command.
	Name string
	// Aliases of the command.
	Aliases []string
	// A short description of the command (one line).
	HelpShort string
	// A long description of the command.
	HelpLong string
	// An instance of the command.
	Command any
}

// Adds a command to the registry.
func (r *Registry) Add(entry RegistryEntry) {
	r.entries = append(r.entries, &entry)
	r.entryMap[Normalize(entry.Name)] = &entry
	if entry.Aliases != nil {
		for _, alias := range entry.Aliases {
			r.entryMap[Normalize(alias)] = &entry
		}
	}
}

// Returns all commands registered to this registry.
func (r Registry) Entries() []*RegistryEntry {
	return r.entries
}

// Returns all commands that match the partial name.
func (r Registry) Matches(namePartial string) []*RegistryEntry {
	name := Normalize(namePartial)

	if entry, ok := r.entryMap[name]; ok {
		return []*RegistryEntry{entry}
	}

	if name == "" {
		return []*RegistryEntry{}
	}

	matchMap := make(map[string]*RegistryEntry)
	for key, entry := range r.entryMap {
		if strings.HasPrefix(key, name) {
			matchMap[entry.Name] = entry
		}
	}

	matches := make([]*RegistryEntry, 0, len(matchMap))
	for _, entry := range matchMap {
		matches = append(matches, entry)
	}

	return matches
}

// Returns the entry which matches the name only if one entry does.
func (r Registry) EntryFor(namePartial string) *RegistryEntry {
	matches := r.Matches(namePartial)
	if len(matches) == 1 {
		return matches[0]
	}
	return nil
}

// Gets an instance of a command with the given name, or nil if non could be found.
func (r Registry) Get(namePartial string) any {
	entry := r.EntryFor(namePartial)
	if entry == nil {
		return nil
	}
	return cloneDefault(entry.Command)
}

// Returns whether the registry has a command with the given name.
func (r Registry) Has(namePartial string) bool {
	return r.EntryFor(namePartial) != nil
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

// An importer that can apply data to a target before Capture is executed.
type CaptureImporter func(data []byte, target any) error

// Captures a command from the options and returns it. The first argument in the options is expected to be the name of the command. If no arguments are given the default "" command is used.
// The remaining arguments are used to populate the value.
// If no arguments are specified beyond the name then interactive mode is enabled by default.
// Interactive (prompt) can be disabled entirely with "--interactive false".
// Importers are also evaluted, like --json, --xml, and --yaml. The value following is the path to the file to import.
func (r Registry) Capture(opts *Options) (any, error) {
	name := ""

	argsLength := len(opts.Args)
	help := GetArg("help", "", &opts.Args, opts.ArgPrefix, false)
	if argsLength != len(opts.Args) {
		return nil, displayHelp(opts, r, help)
	}

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
