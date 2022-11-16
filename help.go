package cmdgo

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

type helpTemplate struct {
	Options   *Options
	Prop      Property
	ArgPrefix string
	Arg       string
}

func (ht helpTemplate) get() string {
	var out bytes.Buffer
	if err := ht.Options.HelpTemplate.Execute(&out, ht); err != nil {
		return ""
	}
	return out.String()
}

func (ht helpTemplate) formatted(prefixSpaces int, wrapLength int, wrapIndent int) string {
	out := ht.get()
	prefix := strings.Repeat(" ", prefixSpaces)
	indent := strings.Repeat(" ", wrapIndent)
	lines := strings.Split(out, "\n")
	linesOutput := make([]string, 0)

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimLeft(lines[i], "\t ")
		if len(trimmed) == 0 {
			continue
		}
		fullLine := prefix + trimmed
		if len(fullLine) > wrapLength {
			wrapped := []string{trimmed}
			for k := 0; k < len(wrapped); k++ {
				last := wrapped[k]
				last = prefix + last
				if k > 0 {
					last = indent + last
				}
				if len(last) > wrapLength {
					a := last[:wrapLength]
					b := last[wrapLength:]
					lastSpace := strings.LastIndex(a, " ")
					if lastSpace != -1 {
						wrapped[k] = a[:lastSpace]
						b = a[lastSpace:] + b
					}
					wrapped = append(wrapped, b)
				} else {
					wrapped[k] = last
				}
			}
			fullLine = strings.Join(wrapped, "\n")
		}
		linesOutput = append(linesOutput, fullLine)
	}
	return strings.Join(linesOutput, "\n")
}

func displayHelp(opts *Options, registry Registry, help string) error {
	if help == "" {
		displayRootHelp(opts, registry)
	} else {
		entry := registry.EntryFor(help)
		if entry == nil {
			opts.Printf("%s is not a valid command. Valid commands:\n", help)
			displayRootHelp(opts, registry)
		} else {
			return displayEntryHelp(opts, entry)
		}
	}
	return nil
}

func displayRootHelp(opts *Options, registry Registry) {
	entries := registry.EntriesAll()
	maxLength := 0
	for _, entry := range entries {
		if len(entry.Name) > maxLength {
			maxLength = len(entry.Name)
		}
	}
	helpFormat := fmt.Sprintf("%%-%ds  %%s\n", maxLength)
	for _, entry := range entries {
		opts.Printf(helpFormat, entry.Name, entry.HelpShort)
	}
}

func displayEntryHelp(opts *Options, entry *RegistryEntry) error {
	opts.Printf("%s", entry.Name)

	if len(entry.Aliases) > 0 {
		aliases := make([]string, 0)

		for _, alias := range entry.Aliases {
			if alias != "" {
				aliases = append(aliases, alias)
			}
		}

		if len(aliases) > 0 {
			opts.Printf(" (aka ")
			for i, alias := range aliases {
				if i > 0 {
					opts.Printf(", ")
				}
				opts.Printf(alias)
			}
			opts.Printf(")")
		}

	}

	opts.Printf(":\n")

	if entry.HelpLong != "" {
		opts.Printf("  %s\n", entry.HelpLong)
	} else if entry.HelpShort != "" {
		opts.Printf("  %s\n", entry.HelpShort)
	}

	helpTpl := helpTemplate{
		Options: opts,
	}

	type HelpDisplayer func(typ reflect.Type, argPrefix string, depth int) error

	var displayTypeHelp HelpDisplayer

	displayTypeHelp = func(typ reflect.Type, argPrefix string, depth int) error {
		var err error

		value := reflect.New(typ)
		inst := GetInstance(value)

		for _, prop := range inst.PropertyList {
			innerKind := reflect.String
			if prop.IsSlice() || prop.IsArray() || prop.IsMap() {
				innerKind = concreteType(prop.Type.Elem()).Kind()
			} else if prop.IsStruct() {
				innerKind = concreteType(prop.Type).Kind()
			}

			arg := argPrefix + strings.ToLower(prop.Arg)
			argTemplate := prop.getArgTemplate(argPrefix, innerKind, nil)
			argTemplate.Index = opts.ArgStartIndex

			switch {
			case prop.IsArray():
				argTemplate.template = opts.ArgArrayTemplate
			case prop.IsSlice():
				argTemplate.template = opts.ArgSliceTemplate
			case prop.IsStruct():
				argTemplate.template = opts.ArgStructTemplate
			case prop.IsMap():
				argTemplate.template = opts.ArgMapKeyTemplate
			}

			if argTemplate.template != nil {
				arg, err = argTemplate.get()
				if err != nil {
					return err
				}
			}

			helpTpl.ArgPrefix = argPrefix
			helpTpl.Arg = strings.ToLower(arg)
			helpTpl.Prop = *prop

			opts.Printf("%s%s\n", strings.Repeat(" ", depth*2), prop.Name)
			help := helpTpl.formatted((depth+1)*opts.HelpIndentWidth, opts.HelpWrapWidth, opts.HelpIndentWidth)
			if help != "" {
				opts.Printf("%s\n", help)
			}

			switch {
			case prop.IsArray(), prop.IsSlice():
				err := displayTypeHelp(prop.ConcreteType().Elem(), arg, depth+1)
				if err != nil {
					return err
				}
			case prop.IsStruct():
				err := displayTypeHelp(prop.ConcreteType(), arg, depth+1)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	var err error

	if entry.Command != nil {
		err = displayTypeHelp(reflect.TypeOf(entry.Command), opts.ArgPrefix, 0)
	}

	return err
}
