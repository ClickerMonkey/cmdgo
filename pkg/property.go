package cmdgo

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// A command property parsed from a command struct.
type CommandProperty struct {
	// The current value of the property
	Value reflect.Value
	// The struct field of the property
	Field reflect.StructField
	// If a prompt should be hidden for this property.  ex: `prompt:"-"`
	HidePrompt bool
	// Text to display when prompting the user. ex: `prompt:"Enter value"`
	PromptText string
	// Help text to display for this property if requested by the user. ex: `help:"your help text here"`
	Help string
	// If the default value should be shown to the user. ex: `default:"-"`
	ShowDefault bool
	// The default value in string form. ex: `default`
	Default string
	// Used by strings for min length, numbers for min value (inclusive), or by slices for min length. ex `min:"1"`
	Min *float64
	// Used by strings for max length, numbers for max value (inclusive), or by slices for max length. ex `max:"10.3"`
	Max *float64
	// Specified with the tag `env:"a,b"`
	Env []string
	// Arg name for this property. ex: `arg:"my-flag"`
	Arg string
}

func (prop *CommandProperty) Load() error {
	text := ""
	if len(prop.Env) > 0 {
		for _, env := range prop.Env {
			envValue := os.Getenv(env)
			if envValue != "" {
				text = envValue
				break
			}
		}
	}
	if text == "" && prop.Default != "" {
		text = prop.Default
	}
	if text != "" {
		return SetString(prop.Value, text)
	}
	return nil
}

func (prop *CommandProperty) FromArgs(ctx CommandContext, args []string) error {
	if prop.Arg == "-" {
		return nil
	}

	value := GetArg(prop.Arg, "", args, ctx.ArgPrefix, prop.IsBool())
	if value != "" {
		err := SetString(prop.Value, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func (prop *CommandProperty) Prompt(ctx CommandContext) error {
	if prop.HidePrompt {
		return nil
	}

	currentValue := prop.Value.Interface()

	promptLabel := prop.PromptText
	if prop.ShowDefault {
		promptLabel = fmt.Sprintf("%s (%+v)", promptLabel, currentValue)
	}

	userInput, err := ctx.Prompt(promptLabel + ": ")
	if err != nil {
		return err
	}
	if userInput == ctx.HelpPrompt && prop.Help != "" {
		ctx.DisplayHelp(*prop)
		userInput, err = ctx.Prompt(promptLabel + ": ")
		if err != nil {
			return err
		}
	}

	if userInput == "" && !prop.ShowDefault && IsDefaultValue(currentValue) {
		userInput, err = ctx.Prompt(promptLabel + " [required]: ")
		if err != nil {
			return err
		}
		if userInput == "" {
			return fmt.Errorf("%s is required", prop.Field.Name)
		}
	}

	if userInput != "" {
		err := SetString(prop.Value, userInput)
		if err != nil {
			return err
		}
	}

	return nil
}

func (prop CommandProperty) Validate() error {
	if prop.Min != nil || prop.Max != nil {
		size := prop.Size()
		if prop.Min != nil && size < *prop.Min {
			return fmt.Errorf("%s has a min of %v", prop.Field.Name, *prop.Min)
		}
		if prop.Max != nil && size > *prop.Max {
			return fmt.Errorf("%s has a max of %v", prop.Field.Name, *prop.Max)
		}
	}
	return nil
}

func (prop CommandProperty) Size() float64 {
	concrete := ConcreteValue(prop.Value)
	kind := concrete.Kind()
	if kind == reflect.Slice || kind == reflect.Array || kind == reflect.String || kind == reflect.Chan || kind == reflect.Map {
		return float64(prop.Value.Len())
	}
	rawValue := concrete.Interface()
	if value, ok := rawValue.(uint); ok {
		return float64(value)
	}
	if value, ok := rawValue.(uint8); ok {
		return float64(value)
	}
	if value, ok := rawValue.(uint16); ok {
		return float64(value)
	}
	if value, ok := rawValue.(uint32); ok {
		return float64(value)
	}
	if value, ok := rawValue.(uint64); ok {
		return float64(value)
	}
	if value, ok := rawValue.(int); ok {
		return float64(value)
	}
	if value, ok := rawValue.(int8); ok {
		return float64(value)
	}
	if value, ok := rawValue.(int16); ok {
		return float64(value)
	}
	if value, ok := rawValue.(int32); ok {
		return float64(value)
	}
	if value, ok := rawValue.(int64); ok {
		return float64(value)
	}
	if value, ok := rawValue.(float32); ok {
		return float64(value)
	}
	if value, ok := rawValue.(float64); ok {
		return float64(value)
	}
	return 0
}

func (prop CommandProperty) IsBool() bool {
	return ConcreteValue(prop.Value).Kind() == reflect.Bool
}

func getCommandProperty(field reflect.StructField, value reflect.Value) CommandProperty {
	prop := CommandProperty{
		Field: field,
		Value: value,
		Env:   make([]string, 0),
	}

	prop.PromptText = field.Name

	if promptText, ok := field.Tag.Lookup("prompt"); ok {
		prop.PromptText = promptText
		if promptText == "-" {
			prop.HidePrompt = true
		}
	}

	if help, ok := field.Tag.Lookup("help"); ok {
		prop.Help = help
	}

	if defaultMode, ok := field.Tag.Lookup("default-mode"); ok {
		prop.ShowDefault = strings.EqualFold(defaultMode, "show")
	}

	if defaultValue, ok := field.Tag.Lookup("default"); ok {
		prop.Default = defaultValue
	}

	if env, ok := field.Tag.Lookup("env"); ok {
		prop.Env = strings.Split(env, ",")
	}

	if arg, ok := field.Tag.Lookup("arg"); ok {
		prop.Arg = Normalize(arg)
	} else {
		prop.Arg = prop.Field.Name
	}

	if min, ok := field.Tag.Lookup("min"); ok {
		minInt, err := strconv.ParseFloat(min, 64)
		if err == nil {
			prop.Min = &minInt
		} else {
			panic(fmt.Sprintf("min of %s is not a valid float64", field.Name))
		}
	}

	if max, ok := field.Tag.Lookup("max"); ok {
		maxInt, err := strconv.ParseFloat(max, 64)
		if err == nil {
			prop.Max = &maxInt
		} else {
			panic(fmt.Sprintf("max of %s is not a valid float64", field.Name))
		}
	}

	return prop
}
