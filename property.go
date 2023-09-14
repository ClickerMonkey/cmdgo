package cmdgo

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
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
	// If the prompt can contain multiple lines and we only stop prompting on an empty line.ex: `prompt-options:"multi"`
	PromptMulti bool
	// If the prompt should ask before it starts to populate a complex type (default true). ex: `prompt-options:"start:"` or `prompt-options:"start:Do you have any favorite numbers (y/n)?"`
	PromptStart string
	// If the prompt should ask before it starts to populate a complex type (default true). ex: `prompt-options:"end:"` or `prompt-options:"end:Thank you for your favorite numbers."`
	PromptEnd string
	// The text to display when questioning for more. ex: `prompt-options:"more:More?"`
	PromptMore string
	// If we should prompt only when the current value is an empty value (not loaded by env, flags, or prompt). ex: `prompt-options:"empty"`
	PromptEmpty bool
	// If the user input should be hidden for this property. ex: `prompt-options:"hidden"`
	InputHidden bool
	// How many tries to get the input. Overrides Context settings. ex: `prompt-options:"tries:4"`
	PromptTries int
	// If we should verify the input by reprompting. ex: `prompt-options:"verify"`
	PromptVerify bool
	// If the property should reprompt given slice and map values. ex `prompt-options="reprompt"`
	Reprompt bool
	// Help text to display for this property if requested by the user. ex: `help:"your help text here"`
	Help string
	// If the default value should be shown to the user. ex: `default-mode:"hide"`
	HideDefault bool
	// Default text to display to override the text version of the current value. ex: `default-text:"***"`
	DefaultText string
	// The default value in string form. ex: `default`
	Default string
	// A regular expression to match.
	Regex string
	// A comma delimited map of acceptable values or a map of key/value pairs. ex: `options:"a,b,c"` or `options:"a:1,b:2,c:3"`
	Choices PromptChoices
	// Used by strings for min length, numbers for min value (inclusive), or by slices for min length. ex `min:"1"`
	Min *float64
	// Used by strings for max length, numbers for max value (inclusive), or by slices for max length. ex `max:"10.3"`
	Max *float64
	// Specified with the tag `env:"a,b"`
	Env []string
	// Arg name for this property. Defaults to the field name. ex: `arg:"my-flag"`
	Arg string
	// Flags that represent how
	Flags Flags[PropertyFlags]
}

// Flags which are set on a property during capture.
type PropertyFlags uint

const (
	// The property has not been changed.
	PropertyFlagNone PropertyFlags = (1 << iota) >> 1
	// The property has had a value populated from arguments.
	PropertyFlagArgs
	// The property has had a value populated from prompting.
	PropertyFlagPrompt
	// The property has had a value populated from environment variables.
	PropertyFlagEnv
	// The property has had a value populated from the `default:""` struct tag.
	PropertyFlagDefault
)

// Returns whether this property can have its state loaded from environment variables
// or default tags.
func (prop Property) CanLoad() bool {
	return !prop.IsIgnored() && prop.IsSimple() && prop.IsDefault()
}

// Loads the initial value of the property from environment variables
// or default tags specified on the struct fields.
func (prop *Property) Load(opts *Options) error {
	if !prop.CanLoad() {
		return nil
	}

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
		return prop.Set(opts, text, flag)
	}
	return nil
}

// Returns whether this property can have its state loaded from arguments.
func (prop Property) CanFromArgs() bool {
	return prop.Arg != "-" && !prop.IsIgnored()
}

func (prop *Property) getArgValue(opts *Options) ArgValue {
	candidate := prop.Value
	if candidate.CanAddr() {
		candidate = candidate.Addr()
	}
	if argValue, ok := candidate.Interface().(ArgValue); ok {
		return argValue
	}
	return nil
}

func (prop *Property) argValue(opts *Options) (bool, error) {
	argValue := prop.getArgValue(opts)
	if argValue != nil {
		return true, argValue.FromArgs(opts, prop, func(arg string, defaultValue string) string {
			return GetArg("", defaultValue, &opts.Args, arg, prop.IsBool())
		})
	}
	return false, nil
}

// Loads value of the property from args if it can and it exists.
func (prop *Property) FromArgs(opts *Options) error {
	if !prop.CanFromArgs() {
		return nil
	}

	if argHandled, err := prop.argValue(opts); argHandled || err != nil {
		return err
	}

	if promptHandled, err := prop.promptValue(opts); promptHandled || err != nil {
		return err
	}

	switch {
	case prop.IsSimple():
		return prop.fromArgsSimple(opts)
	case prop.IsStruct():
		return prop.fromArgsStruct(opts)
	case prop.IsSlice():
		return prop.fromArgsSlice(opts)
	case prop.IsArray():
		return prop.fromArgsArray(opts)
	case prop.IsMap():
		return prop.fromArgsMap(opts)
	}

	return nil
}

func (prop *Property) fromArgsSimple(opts *Options) error {
	value := GetArg(prop.Arg, "", &opts.Args, opts.ArgPrefix, prop.IsBool())
	if value != "" {
		return prop.Set(opts, value, PropertyFlagArgs)
	}

	return nil
}

func (prop Property) promptStart(opts *Options) (bool, error) {
	if prop.HidePrompt {
		return true, nil
	}
	if prop.PromptStart == "-" {
		return true, nil
	}
	if start, err := opts.PromptStart(prop); !start || err != nil {
		return false, err
	}

	return true, nil
}

func (prop Property) promptEnd(opts *Options) error {
	if prop.HidePrompt {
		return nil
	}
	if prop.PromptEnd == "" {
		return nil
	}

	return opts.PromptEnd(prop)
}

func (prop Property) promptMore(opts *Options) (bool, error) {
	if prop.PromptMore == "" {
		return true, nil
	}
	if more, err := opts.PromptMore(prop); !more || err != nil {
		return false, err
	}

	return true, nil
}

func (prop *Property) getPromptValue(opts *Options) PromptCustom {
	candidate := prop.Value
	if candidate.CanAddr() {
		candidate = candidate.Addr()
	}
	if promptValue, ok := candidate.Interface().(PromptCustom); ok {
		return promptValue
	}
	return nil
}

func (prop *Property) promptValue(opts *Options) (bool, error) {
	promptValue := prop.getPromptValue(opts)
	if promptValue != nil {
		return true, promptValue.Prompt(opts, prop)
	}
	return false, nil
}

func (prop *Property) fromArgsStruct(opts *Options) error {
	start, err := prop.promptStart(opts)
	if !start {
		return err
	}

	value := prop.Value
	if prop.IsOptional() && value.IsNil() {
		value = reflect.New(value.Type().Elem())
	}

	argPrefix := opts.ArgPrefix
	defer func() {
		opts.ArgPrefix = argPrefix
	}()

	structTemplate := prop.getArgTemplate(argPrefix, reflect.Struct, opts.ArgStructTemplate)

	prefix, err := structTemplate.get()
	if err != nil {
		return err
	}

	flags, err := captureValue(opts, *prop, value, prefix)
	if err != nil {
		return err
	}

	prop.Flags.Set(flags.value)

	if prop.IsOptional() && !flags.IsEmpty() {
		prop.Value.Set(value)
	}

	err = prop.promptEnd(opts)
	if err != nil {
		return err
	}

	return nil
}

func (prop *Property) fromArgsSlice(opts *Options) error {
	start, err := prop.promptStart(opts)
	if !start {
		return err
	}

	value := prop.Value
	sliceType := concreteType(value.Type())
	if value.IsNil() {
		value = initializeType(value.Type())
	}
	slice := concreteValue(value)

	elementType := sliceType.Elem()
	argPrefix := opts.ArgPrefix
	promptContext := opts.PromptContext
	defer func() {
		opts.ArgPrefix = argPrefix
		opts.PromptContext = promptContext
	}()

	length := slice.Len()

	elementTemplate := prop.getArgTemplate(argPrefix, concreteType(elementType).Kind(), opts.ArgSliceTemplate)

	additionalValues := !prop.HidePrompt

	if (opts.RepromptSliceElements || prop.Reprompt) && opts.CanPrompt() {
		opts.PromptContext.Reprompt = true

		for i := 0; i < length && additionalValues; i++ {
			elementTemplate.Index = i + opts.ArgStartIndex
			elementPrefix, err := elementTemplate.get()
			if err != nil {
				return err
			}

			opts.PromptContext.forSlice(i)

			loaded, err := captureValue(opts, *prop, slice.Index(i), elementPrefix)
			keep := err != ErrDiscard
			if err != nil && keep {
				return err
			}

			if keep {
				if loaded.IsEmpty() && (prop.Min == nil || length+1 >= int(*prop.Min)) && !opts.CanPrompt() {
					break
				}

				prop.Flags.Set(loaded.value)

				if prop.Max != nil && length >= int(*prop.Max) {
					break
				}
			}

			if prop.Min == nil || length >= int(*prop.Min) {
				more, err := prop.promptMore(opts)
				if err != nil {
					return err
				}
				if !more {
					additionalValues = false
				}
			}
		}

		opts.PromptContext.Reprompt = false
	}

	for additionalValues {
		elementTemplate.Index = length + opts.ArgStartIndex
		elementPrefix, err := elementTemplate.get()
		if err != nil {
			return err
		}

		opts.PromptContext.forSlice(length)

		element, loaded, err := captureType(opts, *prop, elementType, elementPrefix)
		keep := err != ErrDiscard
		if err != nil && keep {
			return err
		}

		if keep {
			if loaded.IsEmpty() && (prop.Min == nil || length+1 >= int(*prop.Min)) && !opts.CanPrompt() {
				break
			}

			prop.Flags.Set(loaded.value)
			slice = reflect.Append(slice, element)
			length = slice.Len()

			if prop.Max != nil && length >= int(*prop.Max) {
				break
			}
		}

		if prop.Min == nil || length >= int(*prop.Min) {
			more, err := prop.promptMore(opts)
			if err != nil {
				return err
			}
			if !more {
				additionalValues = false
			}
		}
	}

	if length > 0 {
		setConcrete(prop.Value, slice)
	}

	err = prop.promptEnd(opts)
	if err != nil {
		return err
	}

	return nil
}

func (prop *Property) fromArgsArray(opts *Options) error {
	start, err := prop.promptStart(opts)
	if !start {
		return err
	}

	value := prop.Value
	arrayType := concreteType(value.Type())
	if value.Kind() == reflect.Pointer && value.IsNil() {
		value = initializeType(value.Type())
	}
	array := concreteValue(value)

	argPrefix := opts.ArgPrefix
	defer func() {
		opts.ArgPrefix = argPrefix
	}()

	argFlags := Flags[PropertyFlags]{}

	elementTemplate := prop.getArgTemplate(argPrefix, concreteType(arrayType.Elem()).Kind(), opts.ArgArrayTemplate)

	for i := 0; i < arrayType.Len(); i++ {
		elementTemplate.Index = i + opts.ArgStartIndex

		element := initialize(array.Index(i))
		elementPrefix, err := elementTemplate.get()
		if err != nil {
			return err
		}

		loaded, err := captureValue(opts, *prop, element, elementPrefix)
		if err != nil {
			return err
		}

		argFlags.Set(loaded.value)
	}

	prop.Flags.Set(argFlags.value)

	if value != prop.Value && !argFlags.IsEmpty() {
		setConcrete(prop.Value, array)
	}

	err = prop.promptEnd(opts)
	if err != nil {
		return err
	}

	return nil
}

func (prop *Property) fromArgsMap(opts *Options) error {
	start, err := prop.promptStart(opts)
	if !start {
		return err
	}

	value := prop.Value
	mapType := concreteType(value.Type())
	keyType := mapType.Key()
	valueType := mapType.Elem()
	if value.IsNil() {
		value = initializeType(value.Type())
	}
	mp := concreteValue(value)

	argPrefix := opts.ArgPrefix
	promptContext := opts.PromptContext
	defer func() {
		opts.ArgPrefix = argPrefix
		opts.PromptContext = promptContext
	}()

	argFlags := Flags[PropertyFlags]{}
	length := mp.Len()

	keyTemplate := prop.getArgTemplate(argPrefix, concreteType(keyType).Kind(), opts.ArgMapKeyTemplate)
	valueTemplate := prop.getArgTemplate(argPrefix, concreteType(valueType).Kind(), opts.ArgMapValueTemplate)

	additionalValues := !prop.HidePrompt

	if (opts.RepromptMapValues || prop.Reprompt) && opts.CanPrompt() {
		opts.PromptContext.Reprompt = true

		itr := mp.MapRange()
		for itr.Next() {
			mapKey := itr.Key()
			mapValue := pointerOf(itr.Value()).Elem()

			opts.PromptContext.forMapValue(mapKey.Interface())

			valueLoaded, err := captureValue(opts, *prop, mapValue, "")
			valueKeep := err != ErrDiscard
			if err != nil && valueKeep {
				return err
			}

			if valueKeep {
				mp.SetMapIndex(mapKey, mapValue)
				argFlags.Set(valueLoaded.value)
				length = mp.Len()

				if prop.Max != nil && length >= int(*prop.Max) {
					break
				}

				if prop.Min == nil || length >= int(*prop.Min) {
					more, err := prop.promptMore(opts)
					if err != nil {
						return err
					}
					if !more {
						additionalValues = false
						break
					}
				}
			}
		}

		opts.PromptContext.Reprompt = false
	}

	for additionalValues {
		keyTemplate.Index = length + opts.ArgStartIndex
		valueTemplate.Index = length + opts.ArgStartIndex

		keyPrefix, err := keyTemplate.get()
		if err != nil {
			return err
		}

		opts.PromptContext.forMapKey()

		key, keyLoaded, err := captureType(opts, *prop, keyType, keyPrefix)
		keyKeep := err != ErrDiscard
		if err != nil && keyKeep {
			return err
		}

		if keyKeep {
			if keyLoaded.IsEmpty() && (prop.Min == nil || length+1 >= int(*prop.Min)) && !opts.CanPrompt() {
				break
			}

			valuePrefix, err := valueTemplate.get()
			if err != nil {
				return err
			}

			opts.PromptContext.forMapValue(key.Interface())

			value, valueLoaded, err := captureType(opts, *prop, valueType, valuePrefix)
			valueKeep := err != ErrDiscard
			if err != nil && valueKeep {
				return err
			}

			if valueKeep {
				argFlags.Set(keyLoaded.value | valueLoaded.value)
				mp.SetMapIndex(key, value)
				length = mp.Len()

				if prop.Max != nil && length >= int(*prop.Max) {
					break
				}
			}
		}

		if prop.Min == nil || length >= int(*prop.Min) {
			more, err := prop.promptMore(opts)
			if err != nil {
				return err
			}
			if !more {
				additionalValues = false
			}
		}
	}

	prop.Flags.Set(argFlags.value)

	if mp != prop.Value && !argFlags.IsEmpty() {
		setConcrete(prop.Value, mp)
	}

	err = prop.promptEnd(opts)
	if err != nil {
		return err
	}

	return nil
}

type argTemplate struct {
	Prefix   string
	Arg      string
	Index    int
	IsSimple bool
	IsStruct bool
	IsSlice  bool
	IsMap    bool
	IsArray  bool

	template *template.Template
}

func (tpl argTemplate) get() (string, error) {
	var out bytes.Buffer
	if err := tpl.template.Execute(&out, tpl); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (prop Property) getArgTemplate(argPrefix string, kind reflect.Kind, tpl *template.Template) argTemplate {
	return argTemplate{
		template: tpl,
		Prefix:   argPrefix,
		Arg:      prop.Arg,
		IsSimple: !(kind == reflect.Struct || kind == reflect.Array || kind == reflect.Slice || kind == reflect.Map),
		IsSlice:  kind == reflect.Slice,
		IsStruct: kind == reflect.Struct,
		IsArray:  kind == reflect.Array,
		IsMap:    kind == reflect.Map,
	}
}

func captureType(opts *Options, prop Property, typ reflect.Type, argPrefix string) (reflect.Value, Flags[PropertyFlags], error) {
	value := initializeType(typ)
	flags, err := captureValue(opts, prop, value, argPrefix)
	return value, flags, err
}

func captureValue(opts *Options, prop Property, value reflect.Value, argPrefix string) (Flags[PropertyFlags], error) {
	instance := GetSubInstance(value, prop)

	opts.ArgPrefix = argPrefix
	err := instance.Capture(opts)
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
func (prop *Property) Prompt(opts *Options) error {
	if !prop.CanPrompt() {
		return nil
	}

	if prop.getPromptValue(opts) != nil {
		return nil
	}

	if !opts.CanPrompt() {
		return nil
	}

	switch {
	case prop.IsSimple():
		return prop.promptSimple(opts)
	}

	return nil
}

type promptTemplate struct {
	Prop          Property
	PromptText    string
	DefaultText   string
	IsDefault     bool
	CurrentValue  any
	CurrentText   any
	HideDefault   bool
	PromptCount   int
	HasHelp       bool
	HelpText      string
	AfterHelp     bool
	Context       PromptContext
	InvalidChoice int
	InvalidFormat int
	Verify        bool
	InvalidVerify int

	template *template.Template
}

func (tpl promptTemplate) get() (string, error) {
	var out bytes.Buffer
	if err := tpl.template.Execute(&out, tpl); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (tpl *promptTemplate) updateStatus(status PromptStatus) {
	tpl.AfterHelp = status.AfterHelp
	tpl.PromptCount = status.PromptCount
	tpl.Verify = status.Verify
	tpl.InvalidVerify = status.InvalidVerify
	tpl.InvalidChoice = status.InvalidChoice
	tpl.InvalidFormat = status.InvalidFormat
}

func (prop Property) getPromptTemplate(promptContext PromptContext, tpl *template.Template) promptTemplate {
	currentValue := prop.Value.Interface()
	isDefault := isDefaultValue(currentValue) && !prop.Flags.Is(MatchAny(PropertyFlagDefault))
	currentText := fmt.Sprintf("%+v", currentValue)

	return promptTemplate{
		Prop:         prop,
		PromptText:   prop.PromptText,
		DefaultText:  prop.DefaultText,
		HideDefault:  prop.HideDefault,
		HasHelp:      prop.Help != "",
		HelpText:     prop.Help,
		IsDefault:    isDefault,
		CurrentValue: currentValue,
		CurrentText:  currentText,
		Context:      promptContext,

		template: tpl,
	}
}

func (prop Property) getPromptOnceOptions() PromptOnceOptions {
	return PromptOnceOptions{
		Multi:  prop.PromptMulti,
		Hidden: prop.InputHidden,
	}
}

func (prop *Property) promptSimple(opts *Options) error {
	promptTemplate := prop.getPromptTemplate(opts.PromptContext, opts.PromptTemplate)

	if prop.PromptEmpty {
		if !prop.Flags.Is(MatchAny(PropertyFlagDefault)) && !promptTemplate.IsDefault {
			return nil // user supplied
		}
		if prop.Flags.Is(MatchAny(PropertyFlagArgs | PropertyFlagEnv | PropertyFlagPrompt)) {
			return nil // env/flag/prompt supplied
		}
	}

	tries := opts.RepromptOnInvalid
	if prop.PromptTries > 0 {
		tries = prop.PromptTries
	}

	value, err := opts.Prompt(PromptOptions{
		Prop:     prop,
		Type:     prop.Type,
		Hidden:   prop.InputHidden,
		Verify:   prop.PromptVerify,
		Multi:    prop.PromptMulti,
		Help:     prop.Help,
		Choices:  prop.GetPromptChoices(opts),
		Regex:    prop.Regex,
		Optional: prop.IsOptional() || !promptTemplate.IsDefault,
		Tries:    tries,
		GetPrompt: func(status PromptStatus) (string, error) {
			promptTemplate.updateStatus(status)

			return promptTemplate.get()
		},
	})

	if err != nil {
		return err
	}

	if value != nil {
		prop.Flags.Set(PropertyFlagPrompt)
		prop.Value.Set(reflect.ValueOf(value))
	}

	return nil
}

func (prop Property) Validate(opts *Options) error {
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

	choices := prop.GetPromptChoices(opts)

	if choices != nil && choices.HasChoices() {
		value := prop.ConcreteValue()
		found := false
		for _, option := range choices {
			if isTextuallyEqual(value, option.Value, prop.Type) {
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

func (prop *Property) Set(opts *Options, input string, addFlags PropertyFlags) error {
	choices := prop.GetPromptChoices(opts)
	if choices != nil && choices.HasChoices() {
		converted, err := choices.Convert(input)
		if err != nil {
			return err
		}
		input = converted
	}
	err := SetString(prop.Value, input)
	if err == nil {
		prop.Flags.Set(addFlags)
	}
	return err
}

func (prop *Property) GetPromptChoices(opts *Options) PromptChoices {
	if prop.Choices != nil && prop.Choices.HasChoices() {
		return prop.Choices
	}
	if hasChoices, ok := prop.Value.Interface().(HasChoices); ok {
		return hasChoices.GetChoices(opts, prop)
	}
	return nil
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

func (prop Property) ConcreteType() reflect.Type {
	return concreteType(prop.Type)
}

func (prop Property) HasCustomPromptText() bool {
	return prop.PromptText != prop.Name
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

	prop.PromptStart = fmt.Sprintf("%s?", prop.PromptText)
	prop.PromptMore = fmt.Sprintf("More %s?", prop.PromptText)
	prop.PromptEnd = fmt.Sprintf("End %s", prop.PromptText)

	if promptOptionsText, ok := field.Tag.Lookup("prompt-options"); ok {
		promptOptions := strings.Split(promptOptionsText, ",")
		for _, opt := range promptOptions {
			if opt == "" {
				continue
			}
			keyValue := strings.Split(opt, ":")
			key := strings.ToLower(keyValue[0])
			value := ""
			if len(keyValue) >= 2 {
				value = keyValue[1]
			}
			switch key {
			case "multi":
				prop.PromptMulti = true
			case "reprompt":
				prop.Reprompt = true
			case "start":
				prop.PromptStart = value
			case "end":
				prop.PromptEnd = value
			case "more":
				prop.PromptMore = value
			case "hidden":
				prop.InputHidden = true
			case "verify":
				prop.PromptVerify = true
			case "empty":
				prop.PromptEmpty = true
			case "tries":
				tries, err := strconv.ParseInt(value, 10, 32)
				if err != nil {
					panic(err)
				}
				prop.PromptTries = int(tries)
			}
		}
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

	if regex, ok := field.Tag.Lookup("regex"); ok {
		prop.Regex = regex
	}

	if env, ok := field.Tag.Lookup("env"); ok && env != "" {
		prop.Env = strings.Split(env, ",")
	}

	if arg, ok := field.Tag.Lookup("arg"); ok {
		prop.Arg = Normalize(arg)
	} else {
		prop.Arg = prop.Name
	}

	if min, ok := field.Tag.Lookup("min"); ok {
		if minFloat, err := strconv.ParseFloat(min, 64); err == nil {
			prop.Min = &minFloat
		} else {
			panic(fmt.Sprintf("min of %s is not a valid float64", field.Name))
		}
	}

	if max, ok := field.Tag.Lookup("max"); ok {
		if maxFloat, err := strconv.ParseFloat(max, 64); err == nil {
			prop.Max = &maxFloat
		} else {
			panic(fmt.Sprintf("max of %s is not a valid float64", field.Name))
		}
	}

	prop.Choices = PromptChoices{}

	if options, ok := field.Tag.Lookup("options"); ok && options != "" {
		prop.Choices.FromTag(options, ",", ":")
	}

	return prop
}
