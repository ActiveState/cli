package project

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
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
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	to := v.Type()
	fields := reflect.VisibleFields(to) // we know this is a struct

	for _, f := range fields { // Vars.(OS).Version.Name
		sv := v.FieldByIndex(f.Index)
		if sv.Kind() == reflect.Ptr {
			sv = sv.Elem()
		}
		sto := sv.Type()

		switch sto.Kind() {
		case reflect.Struct:
			m := makeStringMapMap(sto, sv)
			RegisterExpander(strings.ToLower(f.Name), MakeExpanderFuncFromMap(m))

		case reflect.Func:
			//c.RegisterFunc(f.Name, sv.Interface())

		default:
			topLevelLookup[strings.ToLower(f.Name)] = fmt.Sprintf("%v", sv.Interface())
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

func makeStringMapMap(structure reflect.Type, value reflect.Value) map[string]map[string]string {
	m := make(map[string]map[string]string)
	fields := reflect.VisibleFields(structure)

	for _, f := range fields { // Vars.OS.(Version).Name
		subValue := value.FieldByIndex(f.Index)
		if subValue.Kind() == reflect.Ptr {
			subValue = subValue.Elem()
		}
		subType := subValue.Type()

		switch subType.Kind() {
		case reflect.Struct:
			innerMap := make(map[string]string)
			subFields := reflect.VisibleFields(subType)

			for _, sf := range subFields { // Vars.OS.Version.(Name)
				subSubValue := subValue.FieldByIndex(sf.Index)
				innerMap[strings.ToLower(sf.Name)] = fmt.Sprintf("%v", subSubValue.Interface())
			}
			m[strings.ToLower(f.Name)] = innerMap

		default:
			m[strings.ToLower(f.Name)] = map[string]string{"": fmt.Sprintf("%v", value.Interface())}
		}
	}

	return m
}
