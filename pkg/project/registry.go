package project

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
)

// expanderRegistry maps category names to their Expander Func implementations.
var expanderRegistry = map[string]ExpanderFunc{}

var (
	ErrExpandBadName = errs.New("Bad expander name")
	ErrExpandNoFunc  = errs.New("Expander has no handler")
	topLevelLookup   = make(map[string]string)
)

const (
	TopLevelExpanderName = "toplevel"
)

func init() {
	expanderRegistry = map[string]ExpanderFunc{
		"events":             EventExpander,
		"scripts":            ScriptExpander,
		"constants":          ConstantExpander,
		TopLevelExpanderName: TopLevelExpander,
	}
}

func RegisterStruct(val interface{}) error {
	v := reflect.ValueOf(val)
	// deref if needed
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	fields := reflect.VisibleFields(v.Type())

	// Work at depth 1: Vars.[Struct].Struct.Simple
	for _, f := range fields {
		if !f.IsExported() {
			continue
		}

		d1Val := v.FieldByIndex(f.Index)
		if d1Val.Kind() == reflect.Ptr {
			d1Val = d1Val.Elem()
		}

		// If function registration is needed at greater depths, this
		// will need to be reworked (and may not be possible without
		// expansive refactoring).
		switch d1Val.Type().Kind() {
		// Convert type (to map-map) to express advanced control like tag handling.
		case reflect.Struct:
			m := makeEntryMapMap(d1Val)
			name := strings.ToLower(f.Name)
			err := RegisterExpander(name, makeExpanderFuncFromMap(m))
			if err != nil {
				return locale.WrapError(
					err, "project_expand_register_expander_map",
					"Cannot register expander (map)",
				)
			}

		// Expand from function.
		case reflect.Func:
			name := strings.ToLower(f.Name)
			err := RegisterExpander(name, makeExpanderFuncFromFunc(d1Val))
			if err != nil {
				return locale.WrapError(
					err, "project_expand_register_expander_func",
					"Cannot register expander (func)",
				)
			}

		// Format simple value. This is a leaf: Vars.[Simple]
		default:
			topLevelLookup[strings.ToLower(f.Name)] = fmt.Sprintf("%v", d1Val.Interface())
		}
	}

	return nil
}

// RegisterExpander registers an Expander Func for some given handler value. The handler value
// must not effectively be a blank string and the Func must be defined. It is definitely possible
// to replace an existing handler using this function.
func RegisterExpander(handle string, expanderFn ExpanderFunc) error {
	cleanHandle := strings.TrimSpace(handle)
	if cleanHandle == "" {
		return locale.WrapError(ErrExpandBadName, "secrets_expander_err_empty_name")
	} else if expanderFn == nil {
		return locale.WrapError(ErrExpandNoFunc, "secrets_expander_err_undefined")
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
