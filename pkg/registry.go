package cmdgo

var GlobalRegistry = Registry{}

type Registry map[string]any

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

func Register(name string, command any) {
	GlobalRegistry.Add(name, command)
}

func Get(name string) any {
	return GlobalRegistry.Get(name)
}
