package cmdgo

import "testing"

type SimpleCommand struct {
	Message string `default-mode:"show" help:"Help me!"`
}

var _ Command = &SimpleCommand{}

func (cmd *SimpleCommand) Execute(ctx CommandContext) error {
	ctx.Values["result"] = cmd.Message
	return nil
}

func TestSimple(t *testing.T) {
	Register("simple", func() Command { return &SimpleCommand{} })

	tests := []struct {
		args      []string
		result    string
		prompts   map[string]string
		errorText string
		helps     []string
	}{
		{
			args:   []string{},
			result: "",
		},
		{
			args:   []string{"-message", "hi"},
			result: "hi",
		},
		{
			args:   []string{},
			result: "howdy",
			prompts: map[string]string{
				"Message (): ": "howdy",
			},
		},
		{
			args:   []string{"-message", "ok", "-interactive"},
			result: "help!",
			prompts: map[string]string{
				"Message (ok): ": "help!",
			},
			helps: []string{"Help me!"},
		},
	}

	for _, test := range tests {
		actualHelps := []string{}

		ctx := CommandContext{
			HelpPrompt: "help!",
			ArgPrefix:  "-",
			Values:     make(map[string]any),
			Prompt: func(prompt string) (string, error) {
				return test.prompts[prompt], nil
			},
			DisplayHelp: func(prop CommandProperty) {
				actualHelps = append(actualHelps, prop.Help)
			},
		}

		err := Run(ctx, append([]string{"simple"}, test.args...))
		if err != nil {
			if test.errorText == "" {
				t.Fatal(err)
			} else if test.errorText != err.Error() {
				t.Fatalf("Expected error %s but got %s", test.errorText, err.Error())
			}
		} else if ctx.Values["result"] != test.result {
			t.Fatalf("Expected result %s but got %s", test.result, ctx.Values["result"])
		}
		if test.helps != nil && len(test.helps) > 0 {
			if len(test.helps) != len(actualHelps) {
				t.Fatalf("Mismatch in expected helps %d but got %d", len(test.helps), len(actualHelps))
			} else {
				for i := range actualHelps {
					if actualHelps[i] != test.helps[i] {
						t.Fatalf("Expected help %s but got %s", test.helps[i], actualHelps[i])
					}
				}
			}
		}
	}
}
