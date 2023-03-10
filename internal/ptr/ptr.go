package ptr

import (
	"reflect"
)

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type Dereferenceable interface {
	~string | bool
}

func To[T any](v T) *T {
	return &v
}

func Renew[T any](v *T) *T {
	if v == nil {
		return nil
	}
	return &(*v)
}

func DerefInt[D Integer](v *D) D {
	if v != nil {
		return *v
	}

	var x D

	return x - 1
}

func Deref[D Dereferenceable](v *D) D {
	if v != nil {
		return *v
	}

	var x D

	return x
}

// IsNil asserts whether the underlying type is nil, which `interface{} == nil` does not
func IsNil(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsNil()
}
