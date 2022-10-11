package cmdgo

import (
	"errors"
	"reflect"
)

// An error returned from Unmarshal if nil or a non-pointer is passed to Unmarshal.
var InvalidUnmarshalError = errors.New("non-pointer passed to Unmarshal")

// Unmarshal parses the arguments and prompts in ctx and stores the result in the value pointed to by v. If v is nil or not a pointer, Unmarshal returns an InvalidUnmarshalError.
func Unmarshal(ctx *Context, v any) error {
	if v == nil || reflect.ValueOf(v).Kind() != reflect.Pointer {
		return InvalidUnmarshalError
	}
	inst := GetInstance(v)
	return inst.Capture(ctx)
}
