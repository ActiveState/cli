package ptr

import (
	"reflect"
)

// To makes a pointer to a value
func To[T any](v T) *T {
	return &v
}

// From makes a value from a pointer
func From[T any](v *T, fallback T) T {
	if IsNil(v) {
		return fallback
	}
	return *v
}

// Clone create a new pointer with a different memory address than the original, effectively cloning it
func Clone[T any](v *T) *T {
	if v == nil {
		return nil
	}
	t := *v
	return &t
}

// IsNil asserts whether the underlying type is nil, which `interface{} == nil` does not
func IsNil(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsNil()
}
