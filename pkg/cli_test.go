package cmdgo

import (
	"encoding/json"
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

		ctx := NewContextQuiet()
		ctx.ArgPrefix = "-"
		ctx.Prompt = func(prompt string, prop Property) (string, error) {
			return test.prompts[prompt], nil
		}
		ctx.DisplayHelp = func(prop Property) {
			actualHelps = append(actualHelps, prop.Help)
		}

		err := Execute(ctx, append([]string{"simple"}, test.args...))
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
		ctx := NewContextQuiet()
		ctx.ArgPrefix = "-"

		captured, err := Capture(ctx, append([]string{"varied"}, test.args...))
		if err != nil {
			t.Errorf("Test %s failed with error %v", test.name, err)
		} else if !equalsJson(captured, test.expected) {
			t.Errorf("Test %s failed, expected %+v got %+v", test.name, toJson(test.expected), toJson(captured))
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
