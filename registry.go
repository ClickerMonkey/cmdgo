package cmdgo

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// An error returned when no command to capture/execute is given in the arguments.
var ErrNoCommand = errors.New("no command given, try running with --help")

// A map of "commands" by name.
type Registry struct {
	entries  []*RegistryEntry
	entryMap map[string]*RegistryEntry
}

// Creates a new empty registry.
func NewRegistry() Registry {
	return Registry{
		entries:  make([]*RegistryEntry, 0),
		entryMap: make(map[string]*RegistryEntry),
	}
}

// Creates a registry with the given initial entries.
func CreateRegistry(entries []RegistryEntry) Registry {
	r := NewRegistry()
	for _, entry := range entries {
		r.Add(entry)
	}
	return r
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
	// An instance of the command. Either this or Registry should be given.
	Command any
	// A registry of sub commands. Either this or Command should be given.
	Sub Registry
}

// Returns whether the registry is empty.
func (r Registry) IsEmpty() bool {
	return len(r.entries) == 0
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

// Returns all commands registered to this registry and all sub registries.
func (r Registry) EntriesAll() []RegistryEntry {
	all := make([]RegistryEntry, 0, len(r.entries))
	for _, entry := range r.entries {
		all = append(all, *entry)
		if !entry.Sub.IsEmpty() {
			sub := entry.Sub.EntriesAll()
			for _, subEntry := range sub {
				subEntry.Name = entry.Name + " " + subEntry.Name
				all = append(all, subEntry)
			}
		}
	}
	return all
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

// Returns the entry & depth which matches the names if only one entry does - going deep into the entry inner registries until we reach max depth.
func (r Registry) EntryForDeep(namePartials []string) (*RegistryEntry, int) {
	i := 0
	entry := r.EntryFor(namePartials[i])
	for entry != nil {
		if !entry.Sub.IsEmpty() {
			i++
			entry = entry.Sub.EntryFor(namePartials[i])
		} else {
			return entry, i
		}
	}
	return nil, i
}

// Gets an instance of a command with the given name, or nil if non could be found.
func (r Registry) Get(namePartial string) any {
	entry := r.EntryFor(namePartial)
	if entry == nil {
		return nil
	}
	return cloneDefault(entry.Command)
}

// Gets an instance of a command with the given name, or nil if non could be found.
func (r Registry) GetDeep(namePartials []string) (any, int) {
	entry, depth := r.EntryForDeep(namePartials)
	if entry == nil {
		return nil, depth
	}
	return cloneDefault(entry.Command), depth
}

// Returns whether the registry has a command with the given name.
func (r Registry) Has(namePartial string) bool {
	return r.EntryFor(namePartial) != nil
}

// Returns whether the registry has a command with the given name.
func (r Registry) HasDeep(namePartials []string) bool {
	e, _ := r.EntryForDeep(namePartials)
	return e != nil
}

// Executes an executable command based on the given options.
func (r Registry) Execute(opts *Options) error {
	_, err := r.ExecuteReturn(opts)
	return err
}

// Executes an executable command based on the given options and returns it.
func (r Registry) ExecuteReturn(opts *Options) (any, error) {
	cmd, err := r.Capture(opts)
	if err != nil {
		return cmd, err
	}

	if executable, ok := cmd.(Executable); ok {
		return cmd, executable.Execute(opts)
	}

	return cmd, nil
}

// Returns an instance of the command that would be captured based on the given options.
// nil is returned if the options is missing a valid command or is requesting for help.
func (r Registry) Peek(opts *Options) any {
	names := []string{""}
	args := opts.Args[:]
	argsLength := len(args)

	GetArg("help", "", &args, opts.ArgPrefix, false)
	if argsLength != len(args) {
		return nil
	}

	if len(args) == 0 {
		if !r.Has(names[0]) {
			return nil
		}
	} else {
		names = args
	}

	command, _ := r.GetDeep(names)

	return command
}

// An importer that can apply data to a target before Capture is executed.
type CaptureImporter func(data []byte, target any) error

// Captures a command from the options and returns it. The first argument in the options is expected to be the name of the command. If no arguments are given the default "" command is used.
// The remaining arguments are used to populate the value.
// If no arguments are specified beyond the name then interactive mode is enabled by default.
// Interactive (prompt) can be disabled entirely with "--interactive false".
// Importers are also evaluted, like --json, --xml, and --yaml. The value following is the path to the file to import.
func (r Registry) Capture(opts *Options) (any, error) {
	names := []string{""}

	argsLength := len(opts.Args)
	help := GetArg("help", "", &opts.Args, opts.ArgPrefix, false)
	if argsLength != len(opts.Args) {
		return nil, displayHelp(opts, r, help)
	}

	if len(opts.Args) == 0 {
		if !r.Has(names[0]) {
			return nil, ErrNoCommand
		}
	} else {
		names = opts.Args
	}

	command, depth := r.GetDeep(names)

	if command == nil {
		return nil, fmt.Errorf("command not found: %v", names[depth])
	}

	if names[0] != "" {
		opts.Args = opts.Args[depth+1:]
	}

	interactiveDefault := "false"
	if len(opts.Args) == 0 {
		interactiveDefault = "true"
	}

	interactive, _ := strconv.ParseBool(GetArg("interactive", interactiveDefault, &opts.Args, opts.ArgPrefix, true))

	for arg, importer := range CaptureImports {
		path := GetArg(arg, "", &opts.Args, opts.ArgPrefix, false)
		if path != "" {
			imported, err := os.ReadFile(path)
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
