package cmdgo

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// A command property parsed from a command struct.
type Property struct {
	// The current value of the property
	Value reflect.Value
	// The property type
	Type reflect.Type
	// The name of the property
	Name string
	// If a prompt should be hidden for this property.  ex: `prompt:"-"`
	HidePrompt bool
	// Text to display when prompting the user. ex: `prompt:"Enter value"`
	PromptText string
	// If the prompt can contain multiple lines and we only stop prompting on an empty line.ex: `prompt-multi:"true"`
	PromptMulti bool
	// Help text to display for this property if requested by the user. ex: `help:"your help text here"`
	Help string
	// If the default value should be shown to the user. ex: `default-mode:"hide"`
	HideDefault bool
	// Default text to display to override the text version of the current value
	DefaultText string
	// The default value in string form. ex: `default`
	Default string
	// A comma delimited map of acceptable values or a map of key/value pairs. ex: `options:"a,b,c"` or `options:"a:1,b:2,c:3"`
	Options map[string]string
	// Used by strings for min length, numbers for min value (inclusive), or by slices for min length. ex `min:"1"`
	Min *float64
	// Used by strings for max length, numbers for max value (inclusive), or by slices for max length. ex `max:"10.3"`
	Max *float64
	// Specified with the tag `env:"a,b"`
	Env []string
	// Arg name for this property. Defaults to the field name. ex: `arg:"my-flag"`
	Arg string
	// Arg prefix for when this property points to a complex type with inner properties.
	// By default it's the arg of this property with a hyphen appended to it.
	// For no prefix for inner properties specify an empty string. ex: `arg-prefix:""`
	ArgPrefix string
	// Flags that represent how
	Flags Flags[PropertyFlags]
}

type PropertyFlags uint

const (
	PropertyFlagNone PropertyFlags = (1 << iota) >> 1
	PropertyFlagArgs
	PropertyFlagPrompt
	PropertyFlagEnv
	PropertyFlagDefault
)

func (prop Property) Convert(text string) string {
	if prop.Options != nil && len(prop.Options) > 0 {
		key := Normalize(text)
		if converted, ok := prop.Options[key]; ok {
			return converted
		}
		if len(key) > 0 {
			possible := []string{}
			for optionKey, optionValue := range prop.Options {
				if strings.HasPrefix(strings.ToLower(optionKey), key) {
					possible = append(possible, optionValue)
				}
			}
			if len(possible) == 1 {
				return possible[0]
			}
		}
	}
	return text
}

// Returns whether this property can have its state loaded from environment variables
// or default tags.
func (prop Property) CanLoad() bool {
	return !prop.IsIgnored()
}

// Loads the initial value of the property from environment variables
// or default tags specified on the struct fields.
func (prop *Property) Load() error {
	if !prop.CanLoad() {
		return nil
	}

	switch {
	case prop.IsSimple():
		return prop.loadSimple()
		// other loading is done in fromArgsX
	}

	return nil
}

func (prop *Property) loadSimple() error {
	text := ""
	flag := PropertyFlagNone
	if prop.Env != nil && len(prop.Env) > 0 {
		for _, env := range prop.Env {
			envValue := os.Getenv(env)
			if envValue != "" {
				text = envValue
				flag = PropertyFlagEnv
				break
			}
		}
	}
	if text == "" && prop.Default != "" {
		text = prop.Default
		flag = PropertyFlagDefault
	}
	if text != "" {
		return prop.Set(text, flag)
	}
	return nil
}

// Returns whether this property can have its state loaded from arguments.
func (prop Property) CanFromArgs() bool {
	return prop.Arg != "-" && !prop.IsIgnored()
}

// Loads value of the property from args if it can and it exists.
func (prop *Property) FromArgs(ctx Context, args []string) error {
	if !prop.CanFromArgs() {
		return nil
	}

	switch {
	case prop.IsSimple():
		return prop.fromArgsSimple(ctx, args)
	case prop.IsStruct():
		return prop.fromArgsStruct(ctx, args)
	case prop.IsSlice():
		return prop.fromArgsSlice(ctx, args)
	case prop.IsArray():
		return prop.fromArgsArray(ctx, args)
	case prop.IsMap():
		return prop.fromArgsMap(ctx, args)
	}

	return nil
}

func (prop *Property) fromArgsSimple(ctx Context, args []string) error {
	value := GetArg(prop.Arg, "", args, ctx.ArgPrefix, prop.IsBool())
	if value != "" {
		return prop.Set(value, PropertyFlagArgs)
	}

	return nil
}

func (prop *Property) fromArgsStruct(ctx Context, args []string) error {
	value := prop.Value
	if prop.IsOptional() && value.IsNil() {
		value = reflect.New(value.Type().Elem())
	}

	argPrefix := ctx.ArgPrefix
	defer func() {
		ctx.ArgPrefix = argPrefix
	}()

	flags, err := captureValue(ctx, args, *prop, value, argPrefix+prop.ArgPrefix)
	if err != nil {
		return err
	}

	prop.Flags.Set(flags.value)

	if prop.IsOptional() && !flags.IsEmpty() {
		prop.Value.Set(value)
	}

	return nil
}

func (prop *Property) fromArgsSlice(ctx Context, args []string) error {
	value := prop.Value
	sliceType := concreteType(value.Type())
	if value.IsNil() {
		value = initializeType(value.Type())
	}
	slice := concreteValue(value)

	elementType := sliceType.Elem()
	argPrefix := ctx.ArgPrefix
	defer func() {
		ctx.ArgPrefix = argPrefix
	}()

	index := ctx.StartIndex
	length := 0

	for {
		elementPrefix := argPrefix + prop.ArgPrefix + strconv.FormatInt(index, 10)
		if concreteType(elementType).Kind() == reflect.Struct {
			elementPrefix += "-"
		}

		element, loaded, err := captureType(ctx, args, *prop, elementType, elementPrefix)
		if err != nil {
			return err
		}

		if loaded.IsEmpty() && (prop.Min == nil || length+1 >= int(*prop.Min)) {
			break
		}

		prop.Flags.Set(loaded.value)
		slice = reflect.Append(slice, element)
		index++
		length++

		if prop.Max != nil && length >= int(*prop.Max) {
			break
		}
	}

	if index > ctx.StartIndex {
		setConcrete(prop.Value, slice)
	}

	return nil
}

func (prop *Property) fromArgsArray(ctx Context, args []string) error {
	value := prop.Value
	arrayType := concreteType(value.Type())
	if value.Kind() == reflect.Pointer && value.IsNil() {
		value = initializeType(value.Type())
	}
	array := concreteValue(value)

	argPrefix := ctx.ArgPrefix
	defer func() {
		ctx.ArgPrefix = argPrefix
	}()

	index := ctx.StartIndex
	argFlags := Flags[PropertyFlags]{}

	for i := 0; i < arrayType.Len(); i++ {
		element := initialize(array.Index(i))
		elementPrefix := argPrefix + prop.ArgPrefix + strconv.FormatInt(index, 10)
		if concreteKind(element) == reflect.Struct {
			elementPrefix += "-"
		}

		loaded, err := captureValue(ctx, args, *prop, element, elementPrefix)
		if err != nil {
			return err
		}

		argFlags.Set(loaded.value)
		index++
	}

	prop.Flags.Set(argFlags.value)

	if value != prop.Value && !argFlags.IsEmpty() {
		setConcrete(prop.Value, array)
	}

	return nil
}

func (prop *Property) fromArgsMap(ctx Context, args []string) error {
	value := prop.Value
	mapType := concreteType(value.Type())
	keyType := mapType.Key()
	valueType := mapType.Elem()
	if value.IsNil() {
		value = initializeType(value.Type())
	}
	mp := concreteValue(value)

	argPrefix := ctx.ArgPrefix
	defer func() {
		ctx.ArgPrefix = argPrefix
	}()

	argFlags := Flags[PropertyFlags]{}
	length := 0

	for {
		key, keyLoaded, err := captureType(ctx, args, *prop, keyType, argPrefix+prop.ArgPrefix+"key")
		if err != nil {
			return err
		}

		if keyLoaded.IsEmpty() && (prop.Min == nil || length+1 >= int(*prop.Min)) {
			break
		}

		value, valueLoaded, err := captureType(ctx, args, *prop, valueType, argPrefix+prop.ArgPrefix+"value")
		if err != nil {
			return err
		}

		argFlags.Set(keyLoaded.value | valueLoaded.value)
		mp.SetMapIndex(key, value)
		length++

		if prop.Max != nil && length >= int(*prop.Max) {
			break
		}
	}

	prop.Flags.Set(argFlags.value)

	if mp != prop.Value && !argFlags.IsEmpty() {
		setConcrete(prop.Value, mp)
	}

	return nil
}

func captureType(ctx Context, args []string, prop Property, typ reflect.Type, argPrefix string) (reflect.Value, Flags[PropertyFlags], error) {
	value := initializeType(typ)
	flags, err := captureValue(ctx, args, prop, value, argPrefix)
	return value, flags, err
}

func captureValue(ctx Context, args []string, prop Property, value reflect.Value, argPrefix string) (Flags[PropertyFlags], error) {
	instance := GetSubInstance(value, prop)

	ctx.ArgPrefix = argPrefix
	err := instance.Capture(ctx, args)
	if err != nil {
		return Flags[PropertyFlags]{}, err
	}

	important := instance.Flags()
	important.Remove(PropertyFlagDefault | PropertyFlagEnv)

	return important, nil
}

// Returns whether this property can have its state loaded from prompting the user.
func (prop Property) CanPrompt() bool {
	return !prop.HidePrompt && !prop.IsIgnored()
}

// Prompts the user for the value of this property if configured to do so.
func (prop *Property) Prompt(ctx Context) error {
	if !prop.CanPrompt() {
		return nil
	}

	if ctx.Prompt == nil {
		return nil
	}

	switch {
	case prop.IsSimple():
		return prop.promptSimple(ctx)
	case prop.IsStruct():
		return prop.promptStruct(ctx)
	case prop.IsSlice():
		return prop.promptSlice(ctx)
	case prop.IsArray():
		return prop.promptArray(ctx)
	case prop.IsMap():
		return prop.promptMap(ctx)
	}

	return nil
}

func (prop *Property) promptStruct(ctx Context) error {
	return nil
}

func (prop *Property) promptArray(ctx Context) error {
	return nil
}

func (prop *Property) promptSlice(ctx Context) error {
	return nil
}

func (prop *Property) promptMap(ctx Context) error {
	return nil
}

func (prop *Property) promptSimple(ctx Context) error {
	currentValue := prop.Value.Interface()
	isDefault := isDefaultValue(currentValue)

	promptLabel := prop.PromptText
	if prop.DefaultText != "" {
		promptLabel = fmt.Sprintf("%s (%s)", promptLabel, prop.DefaultText)
	} else if !isDefault && !prop.HideDefault {
		promptLabel = fmt.Sprintf("%s (%+v)", promptLabel, currentValue)
	}

	userInput, err := ctx.Prompt(promptLabel+": ", *prop)
	if err != nil {
		return err
	}

	if userInput == ctx.HelpPrompt && ctx.HelpPrompt != "" && prop.Help != "" && ctx.DisplayHelp != nil {
		ctx.DisplayHelp(*prop)
		userInput, err = ctx.Prompt(promptLabel+": ", *prop)
		if err != nil {
			return err
		}
	}

	reprompt := userInput == "" && isDefault && !prop.IsOptional()
	if reprompt {
		userInput, err = ctx.Prompt(promptLabel+" [required]: ", *prop)
		if err != nil {
			return err
		}
		if userInput == "" {
			return fmt.Errorf("%s is required", prop.Name)
		}
	}

	if userInput != "" {
		err := prop.Set(userInput, PropertyFlagPrompt)
		if err != nil {
			return err
		}
	}

	return nil
}

func (prop Property) Validate() error {
	if !prop.IsIgnored() {
		return nil
	}

	if prop.Min != nil || prop.Max != nil {
		size := prop.Size()
		if prop.Min != nil && size < *prop.Min {
			return fmt.Errorf("%s has a min of %v", prop.Name, *prop.Min)
		}
		if prop.Max != nil && size > *prop.Max {
			return fmt.Errorf("%s has a max of %v", prop.Name, *prop.Max)
		}
	}

	if prop.Options != nil && len(prop.Options) > 0 {
		value := prop.ConcreteValue()
		found := false
		for _, optionValue := range prop.Options {
			if isTextuallyEqual(value, optionValue, prop.Type) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%s has an invalid option value: %v", prop.Name, value)
		}
	}
	return nil
}

func (prop Property) Size() float64 {
	kind := concreteKind(prop.Value)
	if kind == reflect.Slice || kind == reflect.Array || kind == reflect.String || kind == reflect.Chan || kind == reflect.Map {
		return float64(prop.Value.Len())
	}

	concrete := concreteValue(prop.Value)
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

func (prop *Property) Set(input string, addFlags PropertyFlags) error {
	converted := prop.Convert(input)
	err := SetString(prop.Value, converted)
	if err == nil {
		prop.Flags.Set(addFlags)
	}
	return err
}

func (prop Property) IsDefault() bool {
	return isDefaultValue(prop.Value.Interface())
}

func (prop Property) IsKind(kind reflect.Kind) bool {
	return concreteKind(prop.Value) == kind
}

func (prop Property) IsKinds(kinds map[reflect.Kind]struct{}) bool {
	_, exists := kinds[concreteKind(prop.Value)]
	return exists
}

func (prop Property) IsNil() bool {
	return prop.Value.IsNil()
}

func (prop Property) IsOptional() bool {
	return prop.Value.Kind() == reflect.Pointer
}

func (prop Property) IsBool() bool {
	return prop.IsKind(reflect.Bool)
}

func (prop Property) IsSlice() bool {
	return prop.IsKind(reflect.Slice)
}

func (prop Property) IsArray() bool {
	return prop.IsKind(reflect.Array)
}

func (prop Property) IsStruct() bool {
	return prop.IsKind(reflect.Struct)
}

func (prop Property) IsMap() bool {
	return prop.IsKind(reflect.Map)
}

func (prop Property) IsSimple() bool {
	return !prop.IsKinds(map[reflect.Kind]struct{}{
		reflect.Array:         {},
		reflect.Slice:         {},
		reflect.Map:           {},
		reflect.Struct:        {},
		reflect.Chan:          {},
		reflect.Func:          {},
		reflect.Interface:     {},
		reflect.UnsafePointer: {},
	})
}

func (prop Property) IsIgnored() bool {
	return prop.IsKinds(map[reflect.Kind]struct{}{
		reflect.Chan:          {},
		reflect.Func:          {},
		reflect.UnsafePointer: {},
	})
}

func (prop Property) ConcreteValue() any {
	concrete := concreteValue(prop.Value)
	rawValue := concrete.Interface()

	return rawValue
}

func getStructProperty(field reflect.StructField, value reflect.Value) Property {
	prop := Property{
		Value: value,
		Type:  field.Type,
		Name:  field.Name,
	}

	prop.PromptText = field.Name

	if promptText, ok := field.Tag.Lookup("prompt"); ok {
		prop.PromptText = promptText
		if promptText == "-" {
			prop.HidePrompt = true
		}
	}

	if promptMulti, ok := field.Tag.Lookup("prompt-multi"); ok {
		prop.PromptMulti, _ = strconv.ParseBool(promptMulti)
	}

	if help, ok := field.Tag.Lookup("help"); ok {
		prop.Help = help
	}

	if defaultMode, ok := field.Tag.Lookup("default-mode"); ok {
		prop.HideDefault = strings.EqualFold(defaultMode, "hide")
	}

	if defaultValue, ok := field.Tag.Lookup("default"); ok {
		prop.Default = defaultValue
	}

	if defaultText, ok := field.Tag.Lookup("default-text"); ok {
		prop.DefaultText = defaultText
	}

	if env, ok := field.Tag.Lookup("env"); ok && env != "" {
		prop.Env = strings.Split(env, ",")
	}

	if arg, ok := field.Tag.Lookup("arg"); ok {
		prop.Arg = Normalize(arg)
	} else {
		prop.Arg = prop.Name
	}

	if argPrefix, ok := field.Tag.Lookup("arg-prefix"); ok {
		prop.ArgPrefix = argPrefix
	} else {
		prop.ArgPrefix = prop.Arg + "-"
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

	if options, ok := field.Tag.Lookup("options"); ok && options != "" {
		prop.Options = make(map[string]string)

		optionList := strings.Split(options, ",")
		for _, option := range optionList {
			keyValue := strings.Split(option, ":")
			key := keyValue[0]
			value := key
			if len(keyValue) > 1 {
				value = keyValue[1]
			}
			prop.Options[Normalize(key)] = value
		}
	}

	return prop
}
