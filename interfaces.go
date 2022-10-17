package cmdgo

// A command is a struct which has properties that can be populated from command-line-arguments,
// prompting the user via stdin, from environment variables, and from configuration files.
// The command properties are evaluated in the order they are defined.

// A command that can be executed after it's data is captured.
type Executable interface {
	// Executes the command
	Execute(opts *Options) error
}

// A dynamic command will have UpdateDynamic invoked before and after every property
// has been gotten from the user/system. This allows the CommandProperties to be
// dynamically changed during data capture OR it allows the state of the command to
// change. For example if a command has two properties and the default of one is based on
type Dynamic interface {
	// The property just updated (or nil if this is the first call) and the map
	// of command properties that can be dynamically updated
	Update(opts *Options, updated *Property, instance *Instance) error
}

// A command can be validated against the current options before it's executed. If an error
// is returned then execution never happens.
type Validator interface {
	Validate(opts *Options) error
}

// A value which has custom prompt handling logic.
type PromptCustom interface {
	Prompt(opts *Options, prop *Property) error
}

// A value which has custom arg handling logic. If this is present, this value
// and any sub values are not handled with arg or prompt logic.
type ArgValue interface {
	FromArgs(opts *Options, prop *Property, getArg func(arg string, defaultValue string) string) error
}

// A value which has custom prompt parsing logic.
type PromptValue interface {
	FromPrompt(opts *Options, value string) error
}

// A value which has user defined options.
type HasChoices interface {
	GetChoices(opts *Options, prop *Property) PromptChoices
}
