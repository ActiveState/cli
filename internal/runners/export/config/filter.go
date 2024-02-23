package config

import (
	"strings"

	"github.com/ActiveState/cli/internal/locale"
)

type Term int

const (
	Dir Term = iota
)

var termLookup = map[Term]string{
	Dir: "dir",
}

func (f Term) String() string {
	for k, v := range termLookup {
		if k == f {
			return v
		}
	}
	return ""
}

func SupportedFilters() []string {
	var supported []string
	for _, v := range termLookup {
		supported = append(supported, v)
	}

	return supported
}

// Filter is the --filter flag for the export config command, it implements captain.FlagMarshaler
type Filter struct {
	terms []Term
}

func (f Filter) String() string {
	output := make([]string, len(f.terms))
	for i, filter := range f.terms {
		output[i] = filter.String()
	}
	return strings.Join(output, ", ")
}

func (f *Filter) Set(value string) error {
	values := strings.Split(value, ",")
	f.terms = make([]Term, 0)
	for _, v := range values {
		for k, termValue := range termLookup {
			if termValue == v {
				f.terms = append(f.terms, k)
				continue
			}
		}
	}

	if len(f.terms) == 0 {
		return locale.NewError("err_invalid_filter", "Invalid filter term specified: '{{.V0}}'; Supported terms: {{.V1}}", value, strings.Join(SupportedFilters(), ", "))
	}
	return nil
}

func (f Filter) Type() string {
	return "filter"
}
