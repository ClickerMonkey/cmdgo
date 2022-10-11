package cmdgo

import (
	"encoding"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type keyValue struct {
	key   string
	value string
}

var _ PromptValue = &keyValue{}

func (kv *keyValue) FromPrompt(ctx *Context, x string) error {
	pair := strings.Split(x, "/")
	kv.key = pair[0]
	kv.value = pair[1]
	return nil
}

type keyValueText struct {
	key   string
	value string
}

var _ encoding.TextUnmarshaler = &keyValueText{}

func (kv *keyValueText) UnmarshalText(data []byte) error {
	pair := strings.Split(string(data), "/")
	kv.key = pair[0]
	kv.value = pair[1]
	return nil
}

func TestPrompt(t *testing.T) {

	tests := []struct {
		name          string
		options       PromptOptions
		expected      any
		expectedError error
		prompts       []string
	}{
		{
			name: "simple",
			options: PromptOptions{
				Prompt: "Name > ",
			},
			prompts: []string{
				"Name > Phil",
			},
			expected: "Phil",
		},
		{
			name: "type -> int",
			options: PromptOptions{
				Prompt: "Name > ",
				Type:   reflect.TypeOf(0),
			},
			prompts: []string{
				"Name > 5",
			},
			expected: 5,
		},
		{
			name: "verify",
			options: PromptOptions{
				Prompt: "Name > ",
				Verify: true,
			},
			prompts: []string{
				"Name > Hi",
				"Name > Hi",
			},
			expected: "Hi",
		},
		{
			name: "verify bad",
			options: PromptOptions{
				Prompt: "Name > ",
				Verify: true,
			},
			prompts: []string{
				"Name > Hi",
				"Name > Ho",
			},
			expectedError: VerifyFailed,
		},
		{
			name: "verify retry",
			options: PromptOptions{
				Prompt: "Name > ",
				Verify: true,
				Tries:  1,
			},
			prompts: []string{
				"Name > Hi",
				"Name > Ho",
				"Name > Hi",
				"Name > Hi",
			},
			expected: "Hi",
		},
		{
			name: "optional string",
			options: PromptOptions{
				Prompt:   "Name > ",
				Optional: true,
			},
			prompts: []string{
				"Name > ",
			},
			expected: nil,
		},
		{
			name: "choices string",
			options: PromptOptions{
				Prompt:  "1 or 2?: ",
				Choices: PromptChoices{"1": "x", "2": "y"},
			},
			prompts: []string{
				"1 or 2?: 1",
			},
			expected: "x",
		},
		{
			name: "choices partial",
			options: PromptOptions{
				Prompt:  "apple or banana?: ",
				Choices: PromptChoices{"apple": "apple", "banana": "banana"},
			},
			prompts: []string{
				"apple or banana?: a",
			},
			expected: "apple",
		},
		{
			name: "choices int",
			options: PromptOptions{
				Prompt:  "1 or 2?: ",
				Type:    reflect.TypeOf(0),
				Choices: PromptChoices{"1": "4", "2": "8"},
			},
			prompts: []string{
				"1 or 2?: 2",
			},
			expected: 8,
		},
		{
			name: "prompt value",
			options: PromptOptions{
				Prompt: "Key/Value: ",
				Type:   reflect.TypeOf(keyValue{}),
			},
			prompts: []string{
				"Key/Value: a/b",
			},
			expected: keyValue{key: "a", value: "b"},
		},
		{
			name: "prompt unmarshal text",
			options: PromptOptions{
				Prompt: "Key/Value: ",
				Type:   reflect.TypeOf(keyValueText{}),
			},
			prompts: []string{
				"Key/Value: a/b",
			},
			expected: keyValueText{key: "a", value: "b"},
		},
		{
			name: "get prompt",
			options: PromptOptions{
				GetPrompt: func(status PromptStatus) (string, error) {
					if status.Verify {
						return "Age (verify): ", nil
					}
					if status.InvalidFormat > 0 {
						return "Age (whole number): ", nil
					}
					return "Age: ", nil
				},
				Tries:  5,
				Type:   reflect.TypeOf(0),
				Verify: true,
			},
			prompts: []string{
				"Age: ",
				"Age (whole number): 33",
				"Age (verify): 33",
			},
			expected: 33,
		},
		{
			name: "regex",
			options: PromptOptions{
				Prompt: "Version > ",
				Regex:  "^\\d{1,4}\\.\\d{1,3}\\.\\d{1,3}(|\\..+)$",
				Tries:  5,
			},
			prompts: []string{
				"Version > a",
				"Version > a.b.c",
				"Version > 2022.12.0",
			},
			expected: "2022.12.0",
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

		actual, err := ctx.Prompt(test.options)

		if err != nil {
			if test.expectedError == nil {
				t.Errorf("Test [%s] unexpected error: %v", test.name, err)
			} else if test.expectedError.Error() != err.Error() {
				t.Errorf("Test [%s] expected '%v' error, got '%v'", test.name, test.expectedError, err)
			}
		} else if toJson(actual) != toJson(test.expected) {
			t.Errorf("Test [%s] expected '%s', got '%s'", test.name, toJson(test.expected), toJson(actual))
		}
	}
}
