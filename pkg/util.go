package cmdgo

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var normalizer, _ = regexp.Compile("[^a-zA-Z0-9]")

func Normalize(x string) string {
	return strings.ToLower(string(normalizer.ReplaceAll([]byte(x), []byte(""))))
}

func SetString(value reflect.Value, s string) error {
	parsed, err := ParseType(value.Type(), s)
	if err != nil {
		return err
	}
	if cast, ok := parsed.(float64); ok {
		value.SetFloat(cast)
	} else if cast, ok := parsed.(bool); ok {
		value.SetBool(cast)
	} else if cast, ok := parsed.(complex128); ok {
		value.SetComplex(cast)
	} else if cast, ok := parsed.(int64); ok {
		value.SetInt(cast)
	} else if cast, ok := parsed.(uint64); ok {
		value.SetUint(cast)
	} else if cast, ok := parsed.(string); ok {
		value.SetString(cast)
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array, reflect.Pointer:
		value.Set(reflect.ValueOf(parsed))
	}
	return nil
}

func ParseType(t reflect.Type, s string) (any, error) {
	switch t.Kind() {
	case reflect.Float32:
		return strconv.ParseFloat(s, 32) // float64, error
	case reflect.Float64:
		return strconv.ParseFloat(s, 64) // float64, error
	case reflect.Bool:
		return strconv.ParseBool(s) // bool, error
	case reflect.Complex64:
		return strconv.ParseComplex(s, 64) // complex128, error
	case reflect.Complex128:
		return strconv.ParseComplex(s, 128) // complex128, error
	case reflect.Int:
		return strconv.ParseInt(s, 10, 64) // int64, error
	case reflect.Int8:
		return strconv.ParseInt(s, 10, 8) // int64, error
	case reflect.Int16:
		return strconv.ParseInt(s, 10, 16) // int64, error
	case reflect.Int32:
		return strconv.ParseInt(s, 10, 32) // int64, error
	case reflect.Int64:
		return strconv.ParseInt(s, 10, 64) // int64, error
	case reflect.Uint:
		return strconv.ParseUint(s, 10, 64) // uint64, error
	case reflect.Uint8:
		return strconv.ParseUint(s, 10, 8) // uint64, error
	case reflect.Uint16:
		return strconv.ParseUint(s, 10, 16) // uint64, error
	case reflect.Uint32:
		return strconv.ParseUint(s, 10, 32) // uint64, error
	case reflect.Uint64:
		return strconv.ParseUint(s, 10, 64) // uint64, error
	case reflect.String:
		return s, nil
	case reflect.Pointer:
		if s == "" {
			return nil, nil
		} else {
			nonNil, err := ParseType(t.Elem(), s)
			return &nonNil, err
		}
	case reflect.Array, reflect.Slice:
		parts := strings.Split(s, ",")
		slice := reflect.MakeSlice(reflect.SliceOf(t.Elem()), 0, len(parts))
		for i := 0; i < len(parts); i++ {
			item, err := ParseType(t.Elem(), parts[i])
			if err != nil {
				return nil, err
			}
			slice = reflect.Append(slice, reflect.ValueOf(item))
		}
		return slice.Interface(), nil
	}
	return nil, nil
}

func ConcreteValue(value reflect.Value) reflect.Value {
	for value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	return value
}

func DefaultValue(t reflect.Type) reflect.Value {
	return reflect.New(t).Elem()
}

func IsDefaultValue(value any) bool {
	defaultValue := DefaultValue(reflect.TypeOf(value)).Interface()

	return ToString(defaultValue) == ToString(value)
}

func ToString(value any) string {
	return fmt.Sprintf("%+v", value)
}

func GetArg(name string, defaultValue string, args []string, argPrefix string, flag bool) string {
	normal := Normalize(name)
	erase := 0
	index := 0
	value := defaultValue
	for index < len(args) {
		arg := args[index]
		if strings.HasPrefix(arg, argPrefix) {
			key := Normalize(arg[len(argPrefix):])
			if key == normal {
				erase = 1
				if index+1 < len(args) {
					value = args[index+1]
					if strings.HasPrefix(value, argPrefix) {
						value = defaultValue
					} else {
						erase = 2
					}
				}
				if flag && value == defaultValue {
					value = "true"
				}
				break
			} else {
				index++
			}
		} else {
			index++
		}
	}

	if erase > 0 {
		args = append(args[0:index], args[index+erase:]...)
	}

	return value
}
