package sliceutils

import (
	"cmp"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/go-openapi/strfmt"
	"golang.org/x/text/unicode/norm"
)

func RemoveFromStrings(slice []string, indexes ...int) []string {
	var out []string
	for i, s := range slice {
		if !intsContain(indexes, i) {
			out = append(out, s)
		}
	}
	return out[:]
}

func GetInt(slice []int, index int) (int, bool) {
	if index < 0 {
		index = len(slice) + index
	}
	if index > len(slice)-1 || index < 0 {
		return -1, false
	}
	return slice[index], true
}

func GetString(slice []string, index int) (string, bool) {
	if index < 0 {
		index = len(slice) + index
	}
	if index > len(slice)-1 || index < 0 {
		return "", false
	}
	// return normalized string
	return norm.NFC.String(slice[index]), true
}

func IntRangeUncapped(in []int, start, end int) []int {
	if end > len(in) {
		end = len(in)
	}
	return in[start:end]
}

func intsContain(ns []int, v int) bool {
	for _, n := range ns {
		if n == v {
			return true
		}
	}
	return false
}

// InsertAt inserts v into data at position i
func InsertStringAt(data []string, i int, v string) []string {
	return append(data[:i], append([]string{v}, data[i:]...)...)
}

func Pop[T any](data []T) (T, []T, error) {
	var t T
	if len(data) == 0 {
		return t, nil, errs.New("Cannot pop from empty slice")
	}

	return data[len(data)-1], data[:len(data)-1], nil
}

func Contains[T comparable](data []T, v T) bool {
	for _, d := range data {
		if d == v {
			return true
		}
	}
	return false
}

func Unique[T comparable](data []T) []T {
	return UniqueByProperty(data, func(d T) any { return d })
}

func UniqueByProperty[T comparable](data []T, uniq func(T) any) []T {
	found := make(map[any]bool)
	out := []T{}
	for _, d := range data {
		v := uniq(d)
		if !found[v] {
			found[v] = true
			out = append(out, d)
		}
	}
	return out
}

func Filter[T any](data []T, f func(T) bool) []T {
	out := []T{}
	for _, d := range data {
		if f(d) {
			out = append(out, d)
		}
	}
	return out
}

func ToLookupMap[T string | int | strfmt.UUID](data []T) map[T]struct{} {
	result := make(map[T]struct{})
	for _, d := range data {
		result[d] = struct{}{}
	}
	return result
}

func ToLookupMapByKey[T any, K string | int | strfmt.UUID](data []T, keyCb func(T) K) map[K]T {
	result := make(map[K]T)
	for _, d := range data {
		result[keyCb(d)] = d
	}
	return result
}

// EqualValues checks if two slices have equal values, regardless of ordering. This does not recurse into nested slices or structs.
func EqualValues[S ~[]E, E cmp.Ordered](a, b S) bool {
	if len(a) != len(b) {
		return false
	}

	lookup := make(map[E]struct{}, len(a))
	for _, e := range a {
		lookup[e] = struct{}{}
	}

	for _, e := range b {
		if _, ok := lookup[e]; !ok {
			return false
		}
	}

	return true
}

// Cast allows casting of a slice of any type to a slice of a specific type.
func Cast[T any](slice []any) ([]T, bool) {
	result := []T{}
	for _, s := range slice {
		v, ok := s.(T)
		if !ok {
			return nil, false
		}
		result = append(result, v)
	}
	return result, true
}
