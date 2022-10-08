package cmdgo

// A command is a struct which has properties that can be populated from command-line-arguments,
// prompting the user via stdin, from environment variables, and from configuration files.
// The command properties are evaluated in the order they are defined.

// A command that can be executed after it's data is captured.
type Executable interface {
	// Executes the command
	Execute(ctx *Context) error
}

// A dynamic command will have UpdateDynamic invoked before and after every property
// has been gotten from the user/system. This allows the CommandProperties to be
// dynamically changed during data capture OR it allows the state of the command to
// change. For example if a command has two properties and the default of one is based on
type Dynamic interface {
	// The property just updated (or nil if this is the first call) and the map
	// of command properties that can be dynamically updated
	Update(ctx *Context, updated *Property, instance *Instance) error
}

// A command can be validated against the current context before it's executed. If an error
// is returned then execution never happens.
type Validator interface {
	Validate(ctx *Context) error
}
