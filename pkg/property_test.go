package cmdgo

import "testing"

func TestConvert(t *testing.T) {
	tests := []struct {
		options  map[string]string
		text     string
		expected string
	}{
		{
			options: map[string]string{
				"a": "a",
				"b": "b",
				"c": "c",
			},
			text:     "A",
			expected: "a",
		},
		{
			options: map[string]string{
				"apple":  "1",
				"blue":   "2",
				"banana": "3",
			},
			text:     "A",
			expected: "1",
		},
		{
			options: map[string]string{
				"apple":  "1",
				"blue":   "2",
				"banana": "3",
			},
			text:     "b",
			expected: "b",
		},
		{
			options: map[string]string{
				"apple":  "1",
				"blue":   "2",
				"banana": "3",
			},
			text:     "ba",
			expected: "3",
		},
	}

	for _, test := range tests {
		prop := Property{
			Options: test.options,
		}

		converted := prop.Convert(test.text)
		if converted != test.expected {
			t.Errorf("Converted %s does not match expected %s", converted, test.expected)
		}
	}
}