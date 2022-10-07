package cmdgo

var Registry = make(map[string]any)

func Register(name string, command any) {
	Registry[Normalize(name)] = command
}

func Get(name string) any {
	if command, ok := Registry[Normalize(name)]; ok {
		copy := cloneDefault(command)
		return copy
	}
	return nil
}
