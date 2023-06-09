package buildscript

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func (s *Script) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	let := make(map[string]interface{})
	for _, assignment := range s.Let.Assignments {
		let[assignment.Key] = assignment.Value
	}
	let["in"] = s.In
	m["let"] = let
	return json.Marshal(m)
}

func (a *Assignment) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m[a.Key] = a.Value
	return json.Marshal(m)
}

func (v *Value) MarshalJSON() ([]byte, error) {
	switch {
	case v.FuncCall != nil:
		return json.Marshal(v.FuncCall)
	case v.List != nil:
		return json.Marshal(v.List)
	case v.Str != nil:
		return json.Marshal(strings.Trim(*v.Str, `"`))
	case v.Assignment != nil:
		return json.Marshal(v.Assignment)
	case v.Object != nil:
		m := make(map[string]interface{})
		for _, assignment := range *v.Object {
			m[assignment.Key] = assignment.Value
		}
		return json.Marshal(m)
	case v.Ident != nil:
		return json.Marshal(v.Ident)
	}
	return json.Marshal(nil)
}

func (f *FuncCall) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	args := make(map[string]interface{})
	for _, argument := range f.Arguments {
		switch {
		case argument.Assignment != nil:
			args[argument.Assignment.Key] = argument.Assignment.Value
		default:
			return nil, errors.New(fmt.Sprintf("Cannot marshal %v (arg %v)", f, argument))
		}
	}
	m[f.Name] = args
	return json.Marshal(m)
}

func (i *In) MarshalJSON() ([]byte, error) {
	switch {
	case i.FuncCall != nil:
		return json.Marshal(i.FuncCall)
	case i.Name != nil:
		return json.Marshal("$" + *i.Name)
	}
	return nil, errors.New(fmt.Sprintf("Cannot marshal %v", i))
}
