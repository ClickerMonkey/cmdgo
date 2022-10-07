package cmdgo

import "golang.org/x/exp/constraints"

type Flags[T constraints.Integer] struct {
	value T
}

func (f *Flags[T]) Set(flags T) {
	f.value = f.value | flags
}
func (f *Flags[T]) Remove(flags T) {
	f.value = f.value & ^flags
}
func (f *Flags[T]) Only(flags T) {
	f.value = f.value & flags
}
func (f *Flags[T]) Toggle(flags T) {
	f.value = f.value ^ flags
}
func (f *Flags[T]) Clear() {
	f.value = 0
}
func (f Flags[T]) IsEmpty() bool {
	return f.value == 0
}
func (f Flags[T]) Get() T {
	return f.value
}
func (f Flags[T]) Is(match Match[T]) bool {
	return match(f.value)
}
