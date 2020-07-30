package config

import (
	"strings"

	"github.com/ActiveState/cli/internal/locale"
)

// Filter is the --filter flag for the export config command, it implements captain.FlagMarshaler
type Filter int

const (
	Dir Filter = iota
)

var filterLookup = map[Filter]string{
	Dir: "dir",
}

func MakeFilter(value string) (*Filter, error) {
	for k, v := range filterLookup {
		if v == value {
			return &k, nil
		}
	}

	return nil, locale.NewError("err_invalid_filter", value, strings.Join(SupportedFilters(), ", "))
}

func (f Filter) String() string {
	for k, v := range filterLookup {
		if k == f {
			return v
		}
	}
	return ""
}

func SupportedFilters() []string {
	var supported []string
	for _, v := range filterLookup {
		supported = append(supported, v)
	}

	return supported
}

func (f *Filter) Set(value string) error {
	for k, v := range filterLookup {
		if v == value {
			*f = k
			return nil
		}
	}

	return locale.NewError("err_invalid_filter", value, strings.Join(SupportedFilters(), ", "))
}

func (f Filter) Type() string {
	return "filter"
}
