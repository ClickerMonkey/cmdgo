package cmdgo

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

type SimpleCommand struct {
	Message string `help:"Help me!"`
}

var _ Executable = &SimpleCommand{}

func (cmd *SimpleCommand) Execute(ctx *Context) error {
	ctx.Values["result"] = cmd.Message
	return nil
}

func TestSimple(t *testing.T) {
	Register("simple", SimpleCommand{})

	tests := []struct {
		args      []string
		result    string
		prompts   map[string]string
		errorText string
		helps     []string
	}{
		{
			args:      []string{},
			result:    "",
			errorText: "Message is required",
		},
		{
			args:      []string{},
			result:    "",
			errorText: "QUIT",
			prompts: map[string]string{
				"Message: ": "quit!",
			},
		},
		{
			args:   []string{"-message", "hi"},
			result: "hi",
		},
		{
			args:   []string{},
			result: "howdy",
			prompts: map[string]string{
				"Message: ": "howdy",
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

		ctx := NewContext().WithArgs(append([]string{"simple"}, test.args...))
		ctx.ArgPrefix = "-"
		ctx.ForcePrompt = true
		ctx.PromptOnce = func(prompt string, options PromptOnceOptions) (string, error) {
			input := test.prompts[prompt]
			if input == ctx.QuitPrompt {
				return input, Quit
			}
			return input, nil
		}
		ctx.DisplayHelp = func(help string, prop *Property) {
			actualHelps = append(actualHelps, help)
		}

		err := Execute(ctx)
		if err != nil {
			if test.errorText == "" {
				t.Error(err)
			} else if test.errorText != err.Error() {
				t.Errorf("Expected error %s but got %s", test.errorText, err.Error())
			}
		} else if ctx.Values["result"] != test.result {
			t.Errorf("Expected result %s but got %s", test.result, ctx.Values["result"])
		}
		if test.helps != nil && len(test.helps) > 0 {
			if len(test.helps) != len(actualHelps) {
				t.Errorf("Mismatch in expected helps %d but got %d", len(test.helps), len(actualHelps))
			} else {
				for i := range actualHelps {
					if actualHelps[i] != test.helps[i] {
						t.Errorf("Expected help %s but got %s", test.helps[i], actualHelps[i])
					}
				}
			}
		}
	}
}

type SimpleStruct struct {
	Prop string
}

type VariedCommand struct {
	SimpleStruct
	String         string
	NilString      *string
	Bool           bool
	NilBool        *bool
	Int            int
	NilInt         *int
	Struct         SimpleStruct
	NilStruct      *SimpleStruct
	IntSlice       []int
	NilIntSlice    *[]int
	StructSlice    []SimpleStruct
	NilStructSlice *[]SimpleStruct
	Array          [2]int
	NilArray       *[2]int
	Map            map[string]int
}

func TestVaried(t *testing.T) {
	Register("varied", VariedCommand{})

	tests := []struct {
		name     string
		args     []string
		expected VariedCommand
	}{
		{
			name:     "empty",
			args:     []string{},
			expected: VariedCommand{},
		},
		{
			name:     "embedded",
			args:     []string{"-prop", "embedded"},
			expected: VariedCommand{SimpleStruct: SimpleStruct{Prop: "embedded"}},
		},
		{
			name:     "string",
			args:     []string{"-string", "a"},
			expected: VariedCommand{String: "a"},
		},
		{
			name:     "nilstring",
			args:     []string{"-nilstring", "nilstring"},
			expected: VariedCommand{NilString: ptrTo("nilstring")},
		},
		{
			name:     "bool",
			args:     []string{"-bool", "1"},
			expected: VariedCommand{Bool: true},
		},
		{
			name:     "nilbool",
			args:     []string{"-nilbool", "1"},
			expected: VariedCommand{NilBool: ptrTo(true)},
		},
		{
			name:     "int",
			args:     []string{"-int", "23"},
			expected: VariedCommand{Int: 23},
		},
		{
			name:     "nilint",
			args:     []string{"-nilint", "23"},
			expected: VariedCommand{NilInt: ptrTo(23)},
		},
		{
			name:     "struct",
			args:     []string{"-struct-prop", "inner"},
			expected: VariedCommand{Struct: SimpleStruct{Prop: "inner"}},
		},
		{
			name:     "nilstruct",
			args:     []string{"-nilstruct-prop", "nilinner"},
			expected: VariedCommand{NilStruct: &SimpleStruct{Prop: "nilinner"}},
		},
		{
			name:     "intslice",
			args:     []string{"-intslice", "1", "-intslice", "2"},
			expected: VariedCommand{IntSlice: []int{1, 2}},
		},
		{
			name:     "structslice",
			args:     []string{"-structslice-1-prop", "1", "-structslice-2-prop", "2"},
			expected: VariedCommand{StructSlice: []SimpleStruct{{Prop: "1"}, {Prop: "2"}}},
		},
		{
			name:     "nilstructslice",
			args:     []string{"-nilstructslice-1-prop", "1"},
			expected: VariedCommand{NilStructSlice: ptrTo([]SimpleStruct{{Prop: "1"}})},
		},
		{
			name:     "array0",
			args:     []string{"-array-1", "1"},
			expected: VariedCommand{Array: [2]int{1, 0}},
		},
		{
			name:     "array1",
			args:     []string{"-array-2", "2"},
			expected: VariedCommand{Array: [2]int{0, 2}},
		},
		{
			name:     "nilarray0",
			args:     []string{"-nilarray-1", "1"},
			expected: VariedCommand{NilArray: ptrTo([2]int{1, 0})},
		},
		{
			name:     "nillarray1",
			args:     []string{"-nilarray-2", "2"},
			expected: VariedCommand{NilArray: ptrTo([2]int{0, 2})},
		},
		{
			name:     "map",
			args:     []string{"-map-key", "a", "-map-value", "1", "-map-key", "b", "-map-value", "2"},
			expected: VariedCommand{Map: map[string]int{"a": 1, "b": 2}},
		},
	}

	for _, test := range tests {
		ctx := NewContext().WithArgs(append([]string{"varied"}, test.args...))
		ctx.ArgPrefix = "-"

		captured, err := Capture(ctx)
		if err != nil {
			t.Errorf("Test [%s] failed with error %v", test.name, err)
		} else if !equalsJson(captured, test.expected) {
			t.Errorf("Test [%s] failed, expected %+v got %+v", test.name, toJson(test.expected), toJson(captured))
		}
	}
}

type ArgValueSimple string

func (my *ArgValueSimple) FromArgs(ctx *Context, prop *Property, getArg func(arg string, defaultValue string) string) error {
	v := getArg("-"+prop.Arg, "")
	if v != "" {
		*my = ArgValueSimple(v + v)
		prop.Flags.Set(PropertyFlagArgs)
	}
	return nil
}

type ArgValueStruct struct {
	Name string
	Age  int
}

func (my *ArgValueStruct) FromArgs(ctx *Context, prop *Property, getArg func(arg string, defaultValue string) string) error {
	v := getArg("-"+prop.Arg, "")
	if v != "" {
		pairs := strings.Split(v, ":")
		if len(pairs) != 2 {
			return fmt.Errorf("%v was not in the Name:Age format.", v)
		}
		my.Name = pairs[0]
		age, err := strconv.ParseInt(pairs[1], 10, 32)
		if err != nil {
			return err
		}
		my.Age = int(age)
		prop.Flags.Set(PropertyFlagArgs)
	}
	return nil
}

type ArgValueCommand struct {
	Simple ArgValueSimple
	Struct ArgValueStruct
}

func TestArgValue(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected ArgValueCommand
	}{
		{
			name:     "empty",
			args:     []string{},
			expected: ArgValueCommand{},
		},
		{
			name: "simple",
			args: []string{"-simple", "x"},
			expected: ArgValueCommand{
				Simple: ArgValueSimple("xx"),
			},
		},
		{
			name: "struct",
			args: []string{"-struct", "Phil:33"},
			expected: ArgValueCommand{
				Struct: ArgValueStruct{
					Name: "Phil",
					Age:  33,
				},
			},
		},
	}

	for _, test := range tests {
		ctx := NewContext().WithArgs(test.args)
		ctx.ArgPrefix = "-"

		actual := ArgValueCommand{}
		err := Unmarshal(ctx, &actual)

		if err != nil {
			t.Errorf("Test [%s] failed with error %v", test.name, err)
		} else if !equalsJson(actual, test.expected) {
			t.Errorf("Test [%s] failed, expected %+v got %+v", test.name, toJson(test.expected), toJson(actual))
		}
	}
}

type PromptValueSimple string

func (my *PromptValueSimple) Prompt(ctx *Context, prop *Property) error {
	v, _ := ctx.PromptOnce(prop.PromptText+": ", prop.getPromptOnceOptions())
	if v != "" {
		*my = PromptValueSimple(v + v)
		prop.Flags.Set(PropertyFlagPrompt)
	}
	return nil
}

type PromptValueStruct struct {
	Name string
	Age  int
}

func (my *PromptValueStruct) Prompt(ctx *Context, prop *Property) error {
	v, _ := ctx.PromptOnce(prop.PromptText+": ", prop.getPromptOnceOptions())
	if v != "" {
		pairs := strings.Split(v, ":")
		if len(pairs) != 2 {
			return fmt.Errorf("%v was not in the Name:Age format.", v)
		}
		my.Name = pairs[0]
		age, err := strconv.ParseInt(pairs[1], 10, 32)
		if err != nil {
			return err
		}
		my.Age = int(age)
		prop.Flags.Set(PropertyFlagPrompt)
	}
	return nil
}

type PromptValueCommand struct {
	Simple PromptValueSimple
	Struct PromptValueStruct
}

func TestPromptValue(t *testing.T) {
	tests := []struct {
		name     string
		expected PromptValueCommand
		prompts  []string
	}{
		{
			name: "empty",
			expected: PromptValueCommand{
				Simple: "aa",
				Struct: PromptValueStruct{
					Name: "Phil",
					Age:  33,
				},
			},
			prompts: []string{
				"Simple: a",
				"Struct: Phil:33",
			},
		},
	}

	for _, test := range tests {
		ctx := NewContext()
		ctx.ForcePrompt = true
		ctx.PromptOnce = func(prompt string, options PromptOnceOptions) (string, error) {
			if len(test.prompts) == 0 {
				return "", fmt.Errorf("No input left for prompt '%s'", prompt)
			}
			line := test.prompts[0]
			test.prompts = test.prompts[1:]
			if strings.HasPrefix(line, prompt) {
				return line[len(prompt):], nil
			} else {
				return "", fmt.Errorf("Prompted '%s', got '%s'", prompt, line)
			}
		}

		actual := PromptValueCommand{}
		err := Unmarshal(ctx, &actual)

		if err != nil {
			t.Errorf("Test [%s] failed with error %v", test.name, err)
		} else if !equalsJson(actual, test.expected) {
			t.Errorf("Test [%s] failed, expected %+v got %+v", test.name, toJson(test.expected), toJson(actual))
		}
	}
}

type RepromptMapCommand struct {
	Map map[string]int
}

func TestRepromptMap(t *testing.T) {
	tests := []struct {
		name     string
		input    RepromptMapCommand
		expected RepromptMapCommand
		prompts  []string
	}{
		{
			name:     "empty",
			input:    RepromptMapCommand{},
			expected: RepromptMapCommand{},
			prompts: []string{
				"Map? (y/n): n",
			},
		},
		{
			name:  "map empty",
			input: RepromptMapCommand{},
			prompts: []string{
				"Map? (y/n): y",
				"Map key: a",
				"Map [a]: 1",
				"More Map? (y/n): n",
			},
			expected: RepromptMapCommand{
				Map: map[string]int{
					"a": 1,
				},
			},
		},
		{
			name: "map reprompt only",
			input: RepromptMapCommand{
				Map: map[string]int{
					"a": 1,
				},
			},
			prompts: []string{
				"Map? (y/n): y",
				"Map [a] (1): 2",
				"More Map? (y/n): n",
			},
			expected: RepromptMapCommand{
				Map: map[string]int{
					"a": 2,
				},
			},
		},
		{
			name: "map reprompt and additional",
			input: RepromptMapCommand{
				Map: map[string]int{
					"a": 1,
				},
			},
			prompts: []string{
				"Map? (y/n): y",
				"Map [a] (1): 2",
				"More Map? (y/n): y",
				"Map key: b",
				"Map [b]: 3",
				"More Map? (y/n): n",
			},
			expected: RepromptMapCommand{
				Map: map[string]int{
					"a": 2,
					"b": 3,
				},
			},
		},
	}

	for _, test := range tests {
		ctx := NewContext()
		ctx.RepromptMapValues = true
		ctx.ForcePrompt = true
		ctx.PromptOnce = func(prompt string, options PromptOnceOptions) (string, error) {
			if len(test.prompts) == 0 {
				return "", fmt.Errorf("No input left for prompt '%s'", prompt)
			}
			line := test.prompts[0]
			test.prompts = test.prompts[1:]
			if strings.HasPrefix(line, prompt) {
				return line[len(prompt):], nil
			} else {
				return "", fmt.Errorf("Prompted '%s', got '%s'", prompt, line)
			}
		}

		err := Unmarshal(ctx, &test.input)
		if err != nil {
			t.Errorf("Test [%s] failed with error %v", test.name, err)
		} else if !equalsJson(test.input, test.expected) {
			t.Errorf("Test [%s] failed, expected %+v got %+v", test.name, toJson(test.expected), toJson(test.input))
		}
	}
}

type RepromptSliceCommand struct {
	Slice []int
}

func TestRepromptSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    RepromptSliceCommand
		expected RepromptSliceCommand
		prompts  []string
	}{
		{
			name:     "empty",
			input:    RepromptSliceCommand{},
			expected: RepromptSliceCommand{},
			prompts: []string{
				"Slice? (y/n): n",
			},
		},
		{
			name:  "slice empty",
			input: RepromptSliceCommand{},
			prompts: []string{
				"Slice? (y/n): y",
				"Slice: 1",
				"More Slice? (y/n): y",
				"Slice: 2",
				"More Slice? (y/n): n",
			},
			expected: RepromptSliceCommand{
				Slice: []int{1, 2},
			},
		},
		{
			name: "slice reprompt 1",
			input: RepromptSliceCommand{
				Slice: []int{1},
			},
			prompts: []string{
				"Slice? (y/n): y",
				"Slice [0] (1): 3",
				"More Slice? (y/n): y",
				"Slice: 2",
				"More Slice? (y/n): n",
			},
			expected: RepromptSliceCommand{
				Slice: []int{3, 2},
			},
		},
		{
			name: "slice reprompt 3",
			input: RepromptSliceCommand{
				Slice: []int{2, 4, 16},
			},
			prompts: []string{
				"Slice? (y/n): y",
				"Slice [0] (2): 0",
				"More Slice? (y/n): y",
				"Slice [1] (4): 1",
				"More Slice? (y/n): y",
				"Slice [2] (16): 2",
				"More Slice? (y/n): y",
				"Slice: 3",
				"More Slice? (y/n): n",
			},
			expected: RepromptSliceCommand{
				Slice: []int{0, 1, 2, 3},
			},
		},
	}

	for _, test := range tests {
		ctx := NewContext()
		ctx.ArgPrefix = "-"
		ctx.RepromptSliceElements = true
		ctx.ForcePrompt = true
		ctx.PromptOnce = func(prompt string, options PromptOnceOptions) (string, error) {
			if len(test.prompts) == 0 {
				return "", fmt.Errorf("No input left for prompt '%s'", prompt)
			}
			line := test.prompts[0]
			test.prompts = test.prompts[1:]
			if strings.HasPrefix(line, prompt) {
				return line[len(prompt):], nil
			} else {
				return "", fmt.Errorf("Prompted '%s', got '%s'", prompt, line)
			}
		}

		err := Unmarshal(ctx, &test.input)
		if err != nil {
			t.Errorf("Test [%s] failed with error %v", test.name, err)
		} else if !equalsJson(test.input, test.expected) {
			t.Errorf("Test [%s] failed, expected %+v got %+v", test.name, toJson(test.expected), toJson(test.input))
		}
	}
}

func equalsJson(a any, b any) bool {
	return toJson(a) == toJson(b)
}

func toJson(a any) string {
	j, _ := json.Marshal(a)
	return string(j)
}

func ptrTo[T any](value T) *T {
	return &value
}
