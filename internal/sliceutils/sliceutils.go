package sliceutils

import (
	"github.com/ActiveState/cli/internal/errs"
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
	if index > len(slice)-1 {
		return -1, false
	}
	return slice[index], true
}

func GetString(slice []string, index int) (string, bool) {
	if index > len(slice)-1 {
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
