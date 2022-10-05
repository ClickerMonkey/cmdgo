package cmdgo

type CommandProvider func() Command

var CommandRegistry = make(map[string]CommandProvider)

func Register(name string, provider CommandProvider) {
	CommandRegistry[Normalize(name)] = provider
}

func Get(name string) Command {
	provider := GetProvider(name)
	if provider == nil {
		return nil
	}
	return (*provider)()
}

func GetProvider(name string) *CommandProvider {
	if provider, ok := CommandRegistry[Normalize(name)]; ok {
		return &provider
	}
	return nil
}
