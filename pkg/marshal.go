package cmdgo

func Unmarshal(ctx *Context, v any) error {
	inst := GetInstance(v)
	return inst.Capture(ctx)
}
