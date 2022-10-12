package cmdgo

import "testing"

func TestUnmarshall(t *testing.T) {
	tests := []struct {
		name          string
		options       *Options
		expected      any
		expectedError error
	}{
		{
			name:    "simple",
			options: NewOptions().WithArgs([]string{"--message", "Hi"}),
			expected: struct {
				Message string
			}{
				Message: "Hi",
			},
		},
	}

	for _, test := range tests {
		actual := cloneDefault(test.expected)
		err := Unmarshal(test.options, actual)
		if err != nil {
			if test.expectedError.Error() != err.Error() {
				t.Errorf("Expected error %s but got %s", test.expectedError.Error(), err.Error())
			}
		} else if !equalsJson(actual, test.expected) {
			t.Errorf("Expected %s but got %s", toJson(test.expected), toJson(actual))
		}

	}
}
