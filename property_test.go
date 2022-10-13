package cmdgo

import "testing"

func TestConvert(t *testing.T) {
	tests := []struct {
		options  PromptChoices
		text     string
		expected string
		invalid  bool
	}{
		{
			options: PromptChoices{
				"a": PromptChoice{Value: "a"},
				"b": PromptChoice{Value: "b"},
				"c": PromptChoice{Value: "c"},
			},
			text:     "A",
			expected: "a",
		},
		{
			options: PromptChoices{
				"apple":  PromptChoice{Value: "1"},
				"blue":   PromptChoice{Value: "2"},
				"banana": PromptChoice{Value: "3"},
			},
			text:     "A",
			expected: "1",
		},
		{
			options: PromptChoices{
				"apple":  PromptChoice{Value: "1"},
				"blue":   PromptChoice{Value: "2"},
				"banana": PromptChoice{Value: "3"},
			},
			text:    "b",
			invalid: true,
		},
		{
			options: PromptChoices{
				"apple":  PromptChoice{Value: "1"},
				"blue":   PromptChoice{Value: "2"},
				"banana": PromptChoice{Value: "3"},
			},
			text:     "ba",
			expected: "3",
		},
	}

	for _, test := range tests {
		converted, err := test.options.Convert(test.text)
		if converted != test.expected {
			t.Errorf("Converted %s does not match expected %s", converted, test.expected)
		} else if (err != nil) != test.invalid {
			t.Errorf("Expected error %v but got %v", test.invalid, err)
		}
	}
}
