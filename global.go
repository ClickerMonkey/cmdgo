package cmdgo

var GlobalRegistry = NewRegistry()

// Adds a command to the global registry.
func Register(entry Entry) {
	GlobalRegistry.Add(entry)
}

// Gets an instance of a command from the global registry with the given name, or nil if non could be found.
func Get(name string) any {
	return GlobalRegistry.Get(name)
}

// Executes an executable command from the global registry based on the given options.
func Execute(opts *Options) error {
	return GlobalRegistry.Execute(opts)
}

// Captures a command from the options and global registry and returns it. See Registry.Capture.
func Capture(opts *Options) (any, error) {
	return GlobalRegistry.Capture(opts)
}

// Peeks a command from the options and global registry and returns it. See Registry.Peek.
func Peek(opts *Options) any {
	return GlobalRegistry.Peek(opts)
}
