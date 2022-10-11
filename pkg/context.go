package cmdgo

import (
	"bufio"
	"encoding"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"text/template"

	"golang.org/x/term"
)

// The error returned when the user requested to quit prompting.
var Quit = errors.New("QUIT")

// The error returned when the user requested to discard the current value being prompted for a complex type.
var Discard = errors.New("DISCARD")

// The error returned when no valid value could be gotten from prompt.
var NoPrompt = errors.New("NOPROMPT")

// The error returned when no valid value could be gotten from prompt.
var VerifyFailed = errors.New("NOVERIFY")

// A dynamic set of variables that commands can have access to during unmarshal, capture, and execution.
type Context struct {
	// A general map of values that can be passed and shared between values being parsed, validated, and updated.
	Values map[string]any

	// The arguments to parse out
	Args []string
	// The prefix all argument names have, to differentiate argument names to values.
	ArgPrefix string
	// The number arrays and maps should start for argument parsing. The number will be in the argument name for arrays or for slices with complex values.
	ArgStartIndex int
	// The template used to generate the argument name/prefix for a struct property.
	ArgStructTemplate *template.Template
	// The template used to generate the argument name/prefix for a slice property.
	ArgSliceTemplate *template.Template
	// The template used to generate the argument name/prefix for an array property.
	ArgArrayTemplate *template.Template
	// The template used to generate the argument name/prefix for map keys
	ArgMapKeyTemplate *template.Template
	// The template used to generate the argument name/prefix for map values
	ArgMapValueTemplate *template.Template

	// The text that should trigger display help for the current prompt.
	HelpPrompt string
	// The text that should trigger prompting to stop immediately and return a Quit error.
	QuitPrompt string
	// The text that should discard the current slice element or map key/value being prompted.
	DiscardPrompt string
	// If prompting should be disabled.
	DisablePrompt bool
	// If prompting should be done even if no input file was given.
	ForcePrompt bool
	// Prompts for a single value. Potentially multiple lines & hidden. If quit or discard prompts are given, the appropriate error is returned.
	PromptOnce func(prompt string, options PromptOnceOptions) (string, error)
	// Prompts the user to start a complex type (struct, slice, array, map) that they can avoid populating.
	PromptStart func(prop Property) (bool, error)
	// The valid options the user can enter which decides if they start prompting for a complex type. The input must match one of the keys (normalized) or prompting will be done repeatedly.
	PromptStartOptions map[string]bool
	// The text to add to the end of the prompt that displays the valid true/false options for the user.
	PromptStartSuffix string
	// Prompts the user to continue populating a complex type (slice, map) that they can avoid adding to.
	PromptMore func(prop Property) (bool, error)
	// The valid options the user can enter which decides if they prompt another value for a complex type. The input must match one of the keys (normalized) or prompting will be done repeatedly.
	PromptMoreOptions map[string]bool
	// The text to add to the end of the prompt that displays the valid true/false options for the user.
	PromptMoreSuffix string
	// A function called at the end of PromptStart and possible PromptMore calls. Can be used to notify user.
	PromptEnd func(prop Property) error
	// A template which converts the current prompt state into a string to send the user.
	PromptTemplate *template.Template
	// The current context of prompts. This metadata is accessible in the PromptTemplate as .Context.
	PromptContext PromptContext
	// If a slice that is prepopulated (via import or developer) should reprompt the values to allow the user to change them.
	RepromptSliceElements bool
	// If a map that is prepopulated (via import or developer) should repomrpt the values to allow the user to change them.
	RepromptMapValues bool
	// How many times the user should be prompted for a valid value.
	RepromptOnInvalid int

	// Displays the requested help to the user.
	DisplayHelp func(help string, prop *Property)

	// Used for displaying and obtaining prompts.
	in       *os.File
	inReader *bufio.Reader
	out      *os.File
}

// A new context which by default has no arguments and does not support prompting.
func NewContext() *Context {
	promptOptions := map[string]bool{
		"y":     true,
		"yes":   true,
		"1":     true,
		"t":     true,
		"true":  true,
		"ok":    true,
		"":      true,
		"n":     false,
		"no":    false,
		"0":     false,
		"f":     false,
		"false": false,
	}

	var ctx *Context

	ctx = &Context{
		Values: make(map[string]any),

		Args:                make([]string, 0),
		ArgPrefix:           "--",
		ArgStartIndex:       1,
		ArgStructTemplate:   newTemplate("{{ .Prefix }}{{ .Arg }}-"),
		ArgSliceTemplate:    newTemplate("{{ .Prefix }}{{ .Arg }}{{ if not .IsSimple }}-{{ .Index }}-{{ end }}"),
		ArgArrayTemplate:    newTemplate("{{ .Prefix }}{{ .Arg }}-{{ .Index }}{{ if not .IsSimple }}-{{ end }}"),
		ArgMapKeyTemplate:   newTemplate("{{ .Prefix }}{{ .Arg }}-key{{ if not .IsSimple }}-{{ end }}"),
		ArgMapValueTemplate: newTemplate("{{ .Prefix }}{{ .Arg }}-value{{ if not .IsSimple }}-{{ end }}"),

		HelpPrompt:         "help!",
		QuitPrompt:         "quit!",
		DiscardPrompt:      "discard!",
		DisablePrompt:      false,
		ForcePrompt:        false,
		PromptStartOptions: promptOptions,
		PromptStartSuffix:  " (y/n): ",
		PromptMoreOptions:  promptOptions,
		PromptMoreSuffix:   " (y/n): ",
		PromptTemplate: newTemplate(`
			{{- .PromptText -}}
			{{- if .Context.IsMapValue }} [{{ .Context.MapKey }}]{{ end }}
			{{- if .Context.IsMapKey }} key{{ end }}
			{{- if and .Context.IsSlice .Context.Reprompt }} [{{ .Context.SliceIndex }}]{{ end }}
			{{- if .DefaultText }} ({{ .DefaultText }})
			{{- else if and (not .IsDefault) (not .HideDefault) }} ({{ .CurrentText }})
			{{- end }}
			{{- if .Verify }} (confirm){{- end }}: `),
		PromptStart: func(prop Property) (bool, error) {
			if !ctx.CanPrompt() {
				return true, nil
			}
			for {
				input, err := ctx.PromptOnce(prop.PromptStart+ctx.PromptStartSuffix, prop.getPromptOnceOptions())
				if err != nil {
					return false, err
				}
				if answer, ok := ctx.PromptStartOptions[strings.ToLower(input)]; ok {
					return answer, nil
				}
			}
		},
		PromptMore: func(prop Property) (bool, error) {
			if !ctx.CanPrompt() {
				return true, nil
			}
			for {
				input, err := ctx.PromptOnce(prop.PromptMore+ctx.PromptMoreSuffix, prop.getPromptOnceOptions())
				if err != nil {
					return false, err
				}
				if answer, ok := ctx.PromptMoreOptions[strings.ToLower(input)]; ok {
					return answer, nil
				}
			}
		},
		PromptEnd: func(prop Property) error {
			if !ctx.CanPrompt() {
				return nil
			}
			return ctx.Printf("%s\n", prop.PromptEnd)
		},
		PromptOnce: func(prompt string, options PromptOnceOptions) (string, error) {
			err := ctx.Printf(prompt)
			if err != nil {
				return "", err
			}
			stop := options.MultiStop + "\n"
			input := ""
			for {
				line := ""
				if options.Hidden {
					bytes, err := term.ReadPassword(int(ctx.in.Fd()))
					if err != nil {
						return "", err
					}
					line = string(bytes)
					ctx.Printf("\n")
				} else {
					line, err = ctx.inReader.ReadString('\n')
					if err != nil && err != io.EOF {
						return "", err
					}
				}
				input += line
				if !options.Multi || line == stop || err != nil {
					input = strings.TrimRight(input, stop)
					break
				}
			}
			input = strings.TrimRight(input, "\n")
			if ctx.QuitPrompt != "" && strings.EqualFold(input, ctx.QuitPrompt) {
				return input, Quit
			}
			if ctx.DiscardPrompt != "" && strings.EqualFold(input, ctx.DiscardPrompt) {
				return input, Discard
			}
			return input, nil
		},
		RepromptOnInvalid:     5,
		RepromptSliceElements: false,
		RepromptMapValues:     false,

		DisplayHelp: func(help string, prop *Property) {
			ctx.Printf("%s\n", help)
		},
	}

	return ctx
}

// Sets the args for the current context. The given slice is unchanged, a copy is retained and updated on the Context during argument parsing.
func (ctx *Context) WithArgs(args []string) *Context {
	ctx.Args = make([]string, len(args))
	copy(ctx.Args, args)
	return ctx
}

// Clears all from the current context.
func (ctx *Context) ClearArgs() *Context {
	ctx.Args = []string{}
	return ctx
}

// Sets the files used during prompting for the current context.
func (ctx *Context) WithFiles(in *os.File, out *os.File) *Context {
	ctx.in = in
	ctx.out = out
	ctx.inReader = bufio.NewReader(in)
	return ctx
}

// Clears all files used during prompting, effectively disabling prompting unless ForcePrompt is specified.
func (ctx *Context) ClearFiles() *Context {
	ctx.in = nil
	ctx.out = nil
	ctx.inReader = nil
	return ctx
}

// Enables prompting using standard in & out.
func (ctx *Context) Std() *Context {
	return ctx.WithFiles(os.Stdin, os.Stdout)
}

// Enables argument parsing using the current running programs arguments.
func (ctx *Context) Cli() *Context {
	return ctx.WithArgs(os.Args[1:])
}

// Enables prompting and arguments from the programs stdin, stdout, and args.
func (ctx *Context) Program() *Context {
	return ctx.Std().Cli()
}

// Prints text out to the configured output destination.
func (ctx *Context) Printf(format string, args ...any) error {
	if ctx.out == nil {
		return nil
	}
	_, err := fmt.Fprintf(ctx.out, format, args...)
	return err
}

// Returns whether this context can prompt the user.
func (ctx *Context) CanPrompt() bool {
	return ctx.ForcePrompt || (ctx.in != nil && !ctx.DisablePrompt)
}

// Options used to prompt a user for a value.
type PromptOptions struct {
	// If specified this function is called to compute the text that is displayed to the user during prompting.
	GetPrompt func(status PromptStatus) (string, error)
	// The prompt text to use if GetPrompt is nil.
	Prompt string
	// If the user's input should be hidden (ex: passwords).
	Hidden bool
	// The type that the input should be converted to. If the user provides an invalid format they may be reprompted.
	// By default the type returned is string. If the type implements PromptValue then the FromPrompt function will be called.
	Type reflect.Type
	// If the input value should be verified (prompts again and ensures they match). Verification is only done after we know the value is valid for the type, choices, etc.
	Verify bool
	// If the input collects multiple lines of text and stops on MultiStop (an empty line by default).
	Multi bool
	// The text which stops collection of multiple lines of text.
	MultiStop string
	// How many times we should try to get valid input from the user.
	Tries int
	// Help text to display if they request it.
	Help string
	// If the value is optional, allowing the user to enter nothing. nil is returned in this scenario.
	Optional bool
	// The property being prompted, if any. This is sent to DisplayHelp.
	Prop *Property
	// The valid inputs and automatic translations. The matching and translation is done before converting it to the desired type.
	Choices PromptChoices
}

// Generates the once options from PromptOptions
func (po PromptOptions) toOnce() PromptOnceOptions {
	return PromptOnceOptions{
		Multi:     po.Multi,
		MultiStop: po.MultiStop,
		Hidden:    po.Hidden,
	}
}

// The current status of the prompt, so useful prompt text can be generated.
type PromptStatus struct {
	// If the user is being asked for a value again directly after they asked for help.
	AfterHelp bool
	// The number of times we've prompted for a value and it was not successful.
	PromptCount int
	// How many times input was given for a prompt with choices and it didn't match.
	InvalidChoice int
	// How many times input was given in an improper format.
	InvalidFormat int
	// If this prompt is to verify the user's inut.
	Verify bool
	// How many times an invalid verification happened.
	InvalidVerify int
}

// Prompts the context for a value given PromptOptions.
func (ctx *Context) Prompt(options PromptOptions) (any, error) {
	once := options.toOnce()
	status := PromptStatus{}
	prompt := options.Prompt
	lastError := NoPrompt

	for i := 0; i <= options.Tries; i++ {
		status.PromptCount = i
		if options.GetPrompt != nil {
			prompt, lastError = options.GetPrompt(status)
			if lastError != nil {
				return nil, lastError
			}
		}

		input, err := ctx.PromptOnce(prompt, once)
		if err != nil {
			return nil, err
		}

		if input == ctx.HelpPrompt && ctx.HelpPrompt != "" && options.Help != "" {
			ctx.DisplayHelp(options.Help, options.Prop)

			status.AfterHelp = true
			if options.GetPrompt != nil {
				prompt, lastError = options.GetPrompt(status)
				if lastError != nil {
					return nil, lastError
				}
			}

			input, err = ctx.PromptOnce(prompt, once)
			if err != nil {
				return nil, err
			}

			status.AfterHelp = false
		}

		if input == "" && options.Optional {
			return nil, nil
		}

		parsed := input
		if options.Choices != nil && options.Choices.HasChoices() {
			parsed, err = options.Choices.Convert(input)
			if err != nil {
				status.InvalidChoice++
				lastError = err
				continue
			}
		}

		instance := pointerOf(reflect.ValueOf(parsed))
		if options.Type != nil {
			instance = reflect.New(options.Type)
		}

		if promptValue, ok := instance.Interface().(PromptValue); ok {
			err = promptValue.FromPrompt(ctx, parsed)
			if err != nil {
				status.InvalidFormat++
				lastError = err
				continue
			}
		} else if textUnmarshall, ok := instance.Interface().(encoding.TextUnmarshaler); ok {
			err = textUnmarshall.UnmarshalText([]byte(parsed))
			if err != nil {
				status.InvalidFormat++
				lastError = err
				continue
			}
		} else {
			err = SetString(instance, parsed)
			if err != nil {
				status.InvalidFormat++
				lastError = err
				continue
			}
		}

		if options.Verify {
			status.Verify = true
			if options.GetPrompt != nil {
				prompt, lastError = options.GetPrompt(status)
				if lastError != nil {
					return nil, lastError
				}
			}
			verifyInput, err := ctx.PromptOnce(prompt, once)
			status.Verify = false
			if err != nil {
				return nil, err
			}
			if verifyInput != input {
				status.InvalidVerify++
				lastError = VerifyFailed
				continue
			}
		}

		return instance.Elem().Interface(), nil
	}

	return nil, lastError
}

// Context that is changed during the prompt process.
type PromptContext struct {
	// If the user is currently being prompted for a map key.
	IsMapKey bool
	// If the user is currently being prompted for a map value.
	IsMapValue bool
	// The string representation of the key of the value being prompted when IsMapValue = true.
	MapKey string
	// If the user is currently being prompted for a slice element.
	IsSlice bool
	// The index of the element when being prompted for a slice element.
	SliceIndex int
	// If the current value is part of a slice or map reprompting.
	Reprompt bool
}

func (pc *PromptContext) reset() {
	pc.IsMapKey = false
	pc.MapKey = ""
	pc.IsMapValue = false
	pc.SliceIndex = -1
	pc.IsSlice = false
}

func (pc *PromptContext) forMapKey() {
	pc.reset()
	pc.IsMapKey = true
}

func (pc *PromptContext) forMapValue(key any) {
	pc.reset()
	pc.MapKey = fmt.Sprintf("%+v", key)
	pc.IsMapValue = true
}

func (pc *PromptContext) forSlice(index int) {
	pc.reset()
	pc.IsSlice = true
	pc.SliceIndex = index
}

// Options that can be passed when prompting for a single input.
type PromptOnceOptions struct {
	Multi     bool
	Hidden    bool
	MultiStop string
}

// Creates a parsed template and panics if it's invalid.
func newTemplate(pattern string) *template.Template {
	tpl, err := template.New("").Parse(pattern)
	if err != nil {
		panic(err)
	}
	return tpl
}
