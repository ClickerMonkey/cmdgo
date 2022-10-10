package cmdgo

var GlobalRegistry = Registry{}

func Register(name string, command any) {
	GlobalRegistry.Add(name, command)
}

func Get(name string) any {
	return GlobalRegistry.Get(name)
}

func Execute(ctx *Context) error {
	return GlobalRegistry.Execute(ctx)
}

func Capture(ctx *Context) (any, error) {
	return GlobalRegistry.Capture(ctx)
}
