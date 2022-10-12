package cmdgo

var GlobalRegistry = Registry{}

func Register(name string, command any) {
	GlobalRegistry.Add(name, command)
}

func Get(name string) any {
	return GlobalRegistry.Get(name)
}

func Execute(opts *Options) error {
	return GlobalRegistry.Execute(opts)
}

func Capture(opts *Options) (any, error) {
	return GlobalRegistry.Capture(opts)
}
