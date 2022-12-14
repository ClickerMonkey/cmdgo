package cmdgo

import (
	"errors"
	"strings"
)

// An error returned when input is given to prompt choices and no choices could be determined.
var ErrInvalidConversion = errors.New("invalid conversion")

// A choice when prompting/parsing arg text values.
type PromptChoice struct {
	Text  string
	Value string
}

// A map of inputs to translated values. Matching is done ignoring punctuation and will do partial
// matching if only one choice is a partial match.
type PromptChoices map[string]PromptChoice

// Parses choices from a tag string.
// choices.FromTag("a:1,b:2,c:3", ",", ":") is parsed to {"a":1,"b":2,"c":3}
func (pc PromptChoices) FromTag(tag string, pairDelimiter string, keyValueDelimiter string) {
	keyValueList := strings.Split(tag, pairDelimiter)
	for _, option := range keyValueList {
		keyValue := strings.Split(option, keyValueDelimiter)
		key := keyValue[0]
		value := key
		if len(keyValue) > 1 {
			value = keyValue[1]
		}
		pc.Add(key, value)
	}
}

// Adds an input and translated value to choices.
func (pc PromptChoices) Add(input string, value string) {
	pc[Normalize(input)] = PromptChoice{
		Text:  input,
		Value: value,
	}
}

// Converts the input to a translated value OR returns an InvalidConversion error.
// If choices is empty then the input given is returned. If input partially matches
// exactly one choice (normalized) then its assumed to be that value.
func (pc PromptChoices) Convert(input string) (string, error) {
	if !pc.HasChoices() {
		return input, nil
	}

	key := Normalize(input)
	if converted, ok := pc[key]; ok {
		return converted.Value, nil
	}
	if len(key) > 0 {
		possible := []string{}
		for optionKey, optionValue := range pc {
			if strings.HasPrefix(strings.ToLower(optionKey), key) {
				possible = append(possible, optionValue.Value)
			}
		}
		if len(possible) == 1 {
			return possible[0], nil
		}
	}

	return "", ErrInvalidConversion
}

// Returns whether there are any choices defined.
func (pc PromptChoices) HasChoices() bool {
	return len(pc) > 0
}
