package cmdgo

import "golang.org/x/exp/constraints"

type Match[T constraints.Integer] func(value T) bool

func MatchAll[T constraints.Integer](test T) Match[T] {
	return func(value T) bool {
		return value&test == test
	}
}

func MatchOnly[T constraints.Integer](test T) Match[T] {
	return func(value T) bool {
		return value&test == value
	}
}

func MatchExact[T constraints.Integer](test T) Match[T] {
	return func(value T) bool {
		return value&value == test
	}
}

func MatchAny[T constraints.Integer](test T) Match[T] {
	return func(value T) bool {
		return value&test != 0
	}
}

func MatchNone[T constraints.Integer](test T) Match[T] {
	return func(value T) bool {
		return value&test == 0
	}
}

func MatchEmpty[T constraints.Integer]() Match[T] {
	return func(value T) bool {
		return value&value == 0
	}
}

func MatchNot[T constraints.Integer](not Match[T]) Match[T] {
	return func(value T) bool {
		return !not(value)
	}
}

func MatchAnd[T constraints.Integer](ands ...Match[T]) Match[T] {
	return func(value T) bool {
		for _, and := range ands {
			if !and(value) {
				return false
			}
		}
		return true
	}
}

func MatchOr[T constraints.Integer](ors ...Match[T]) Match[T] {
	return func(value T) bool {
		for _, or := range ors {
			if or(value) {
				return true
			}
		}
		return false
	}
}
