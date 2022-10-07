package cmdgo

import "testing"

func TestGetArg(t *testing.T) {
	tests := []struct {
		name         string
		defaultValue string
		args         []string
		flag         bool
		expected     string
	}{
		{
			name:         "a",
			defaultValue: "",
			args:         []string{},
			flag:         false,
			expected:     "",
		},
		{
			name:         "a",
			defaultValue: "",
			args:         []string{"-a", "hello"},
			flag:         false,
			expected:     "hello",
		},
		{
			name:         "doit",
			defaultValue: "",
			args:         []string{"-doit", "false"},
			flag:         true,
			expected:     "false",
		},
		{
			name:         "doit",
			defaultValue: "",
			args:         []string{"-doit"},
			flag:         true,
			expected:     "true",
		},
		{
			name:         "doit",
			defaultValue: "",
			args:         []string{"-not", "3"},
			flag:         true,
			expected:     "",
		},
		{
			name:         "doit",
			defaultValue: "xx",
			args:         []string{"-not", "3"},
			flag:         false,
			expected:     "xx",
		},
	}

	for _, test := range tests {
		actual := GetArg(test.name, test.defaultValue, test.args, "-", test.flag)
		if actual != test.expected {
			t.Errorf("Expected %s but got %s", test.expected, actual)
		}
	}

}

func TestNormalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "normalize me cap!",
			expected: "normalizemecap",
		},
		{
			input:    "--CAPS",
			expected: "caps",
		},
		{
			input:    "okay_8",
			expected: "okay8",
		},
	}

	for _, test := range tests {
		actual := Normalize(test.input)
		if actual != test.expected {
			t.Errorf("Expected %s but got %s", test.expected, actual)
		}
	}
}

func TestIsDefaultValue(t *testing.T) {
	tests := []struct {
		value     any
		isDefault bool
	}{
		{
			value:     int(0),
			isDefault: true,
		},
		{
			value:     int(1),
			isDefault: false,
		},
		{
			value:     "",
			isDefault: true,
		},
		{
			value:     "a",
			isDefault: false,
		},
		{
			value:     " ",
			isDefault: false,
		},
		{
			value:     struct{}{},
			isDefault: true,
		},
		{
			value:     struct{ inner string }{},
			isDefault: true,
		},
		{
			value:     struct{ inner string }{inner: "no"},
			isDefault: false,
		},
	}

	for _, test := range tests {
		actual := isDefaultValue(test.value)
		if actual != test.isDefault {
			t.Errorf("Expected %v but got %v", test.isDefault, actual)
		}
	}
}
