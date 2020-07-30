package config

import (
	"strings"

	"github.com/ActiveState/cli/internal/locale"
)

// Filter is the --filter flag for the export config command, it implements captain.FlagMarshaler
type Filter int

const (
	Unset Filter = iota
	Dir
)

var filterLookup = map[Filter]string{
	Unset: "unset",
	Dir:   "dir",
}

func (f Filter) String() string {
	for k, v := range filterLookup {
		if k == f {
			return v
		}
	}
	return filterLookup[Unset]
}

func SupportedFilters() []string {
	var supported []string
	for k, v := range filterLookup {
		if k != Unset {
			supported = append(supported, v)
		}
	}

	return supported
}

func (f *Filter) Set(value string) error {
	for k, v := range filterLookup {
		if v == value && k != Unset {
			*f = k
			return nil
		}
	}

	return locale.NewError("err_invalid_filter", value, strings.Join(SupportedFilters(), ", "))
}

func (f Filter) Type() string {
	return "filter"
}
