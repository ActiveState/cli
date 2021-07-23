package project

import (
	"strings"

	"github.com/ActiveState/cli/internal/errs"
)

// expanderRegistry maps category names to their Expander Func implementations.
var expanderRegistry = map[string]ExpanderFunc{}

var (
	ErrExpandBadName = errs.New("Bad expander name")
	ErrExpandNoFunc  = errs.New("Expander has no handler")
)

const TopLevelExpanderName = "toplevel"

func init() {
	expanderRegistry = map[string]ExpanderFunc{
		"platform":           PlatformExpander,
		"events":             EventExpander,
		"scripts":            ScriptExpander,
		"constants":          ConstantExpander,
		"project":            ProjectExpander,
		TopLevelExpanderName: TopLevelExpander,
	}
}

// RegisterExpander registers an Expander Func for some given handler value. The handler value
// must not effectively be a blank string and the Func must be defined. It is definitely possible
// to replace an existing handler using this function.
func RegisterExpander(handle string, expanderFn ExpanderFunc) error {
	cleanHandle := strings.TrimSpace(handle)
	if cleanHandle == "" {
		return errs.Wrap(ErrExpandBadName, "secrets_expander_err_empty_name")
	} else if expanderFn == nil {
		return errs.Wrap(ErrExpandNoFunc, "secrets_expander_err_undefined")
	}
	expanderRegistry[cleanHandle] = expanderFn
	return nil
}

// RegisteredExpander returns the expander registered for the given handle
func RegisteredExpander(handle string) ExpanderFunc {
	if expander, ok := expanderRegistry[handle]; ok {
		return expander
	}
	return nil
}

// IsRegistered returns true if an Expander Func is registered for a given handle/name.
func IsRegistered(handle string) bool {
	_, ok := expanderRegistry[handle]
	return ok
}
