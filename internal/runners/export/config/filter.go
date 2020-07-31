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

// Filters is the --filter flag for the export config command, it implements captain.FlagMarshaler
type Filters struct {
	filters []Filter
}

func (f Filters) String() string {
	output := make([]string, len(f.filters))
	for i, filter := range f.filters {
		output[i] = filter.String()
	}
	return strings.Join(output, ", ")
}

func (f *Filters) Set(value string) error {
	values := strings.Split(value, ",")
	f.filters = make([]Filter, 0)
	for _, v := range values {
		for k, filterString := range filterLookup {
			if filterString == v {
				f.filters = append(f.filters, k)
				continue
			}
		}
	}

	if len(f.filters) == 0 {
		return locale.NewError("err_invalid_filter", value, strings.Join(SupportedFilters(), ", "))
	}
	return nil
}

func (f Filters) Type() string {
	return "filters"
}
