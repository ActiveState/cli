package buildscript

import (
	"errors"
	"fmt"
	"time"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/brunoga/deep"
)

// BuildScript is what we want consuming code to work with. This specifically makes the raw
// presentation private as no consuming code should ever be looking at the raw representation.
// Instead this package should facilitate the use-case of the consuming code through convenience
// methods that are easy to understand and work with.
type BuildScript struct {
	raw *rawBuildScript
}

func init() {
	// Guard against emptyBuildExpression having parsing issues
	if !condition.BuiltViaCI() || condition.InActiveStateCI() {
		err := New().UnmarshalBuildExpression([]byte(emptyBuildExpression))
		if err != nil {
			panic(err)
		}
	}
}

func Create() *BuildScript {
	bs := New()
	// We don't handle unmarshalling errors here, see the init function for that.
	// Since the empty build expression is a constant there's really no need to error check this each time.
	_ = bs.UnmarshalBuildExpression([]byte(emptyBuildExpression))
	return bs
}

func New() *BuildScript {
	bs := &BuildScript{raw: &rawBuildScript{}}
	return bs
}

func (b *BuildScript) AtTime() *time.Time {
	return b.raw.AtTime
}

func (b *BuildScript) SetAtTime(t time.Time) {
	b.raw.AtTime = &t
}

func (b *BuildScript) Equals(other *BuildScript) (bool, error) {
	myBytes, err := b.Marshal()
	if err != nil {
		return false, errs.New("Unable to marshal this buildscript: %s", errs.JoinMessage(err))
	}
	otherBytes, err := other.Marshal()
	if err != nil {
		return false, errs.New("Unable to marshal other buildscript: %s", errs.JoinMessage(err))
	}
	return string(myBytes) == string(otherBytes), nil
}

func (b *BuildScript) Clone() (*BuildScript, error) {
	bb, err := deep.Copy(b)
	if err != nil {
		return nil, errs.Wrap(err, "unable to clone buildscript")
	}
	return bb, nil
}

// FuncCall is the exportable version of funcCall, because we do not want to expose low level buildscript functionality
// outside of the buildscript package.
type FuncCall struct {
	fc *funcCall
}

func (f *FuncCall) MarshalJSON() ([]byte, error) {
	return f.fc.MarshalJSON()
}

// Argument returns the value of the given argument, or nil if it does not exist
// You will still need to cast the value to the correct type.
func (f *FuncCall) Argument(name string) any {
	for _, a := range f.fc.Arguments {
		if a.Assignment == nil || a.Assignment.Key != name {
			continue
		}
		return exportValue(a.Assignment.Value)
	}
	return nil
}

// SetArgument will update the given argument, or add it if it does not exist
func (f *FuncCall) SetArgument(k string, v *value) {
	for i, a := range f.fc.Arguments {
		if a.Assignment == nil || a.Assignment.Key != k {
			continue
		}
		f.fc.Arguments[i].Assignment.Value = v
		return
	}

	// Arg doesn't exist; append it instead
	f.fc.Arguments = append(f.fc.Arguments, &value{Assignment: &assignment{Key: k, Value: v}})

	return
}

// UnsetArgument will remove the given argument, if it exists
func (f *FuncCall) UnsetArgument(k string) {
	for i, a := range f.fc.Arguments {
		if a.Assignment == nil || a.Assignment.Key != k {
			continue
		}
		f.fc.Arguments = append(f.fc.Arguments[:i], f.fc.Arguments[i+1:]...)
		return
	}
}

// Value turns a standard type into a buildscript compatible type
// Intended for use with functions like SetArgument.
func Value[T string | float64 | []string | []float64](inputv T) *value {
	v := &value{}
	switch vt := any(inputv).(type) {
	case string:
		v.Str = &vt
	case float64:
		v.Number = &vt
	case []string:
		strValues := make([]*value, len(vt))
		for i, s := range vt {
			strValues[i] = &value{Str: &s}
		}
		v.List = &strValues
	case []float64:
		numValues := make([]*value, len(vt))
		for i, n := range vt {
			numValues[i] = &value{Number: &n}
		}
		v.List = &numValues
	}

	return v
}

// exportValue takes a raw buildscript value and turns it into an externally consumable one
// Note not all value types are currently fully supported. For example assignments and objects currently are
// passed as the raw type, which can't be cast externally as they are private types.
// We'll want to update these as the use-cases for them become more clear.
func exportValue(v *value) any {
	switch {
	case v.FuncCall != nil:
		if req := parseRequirement(v); req != nil {
			return req
		}
		return &FuncCall{v.FuncCall}
	case v.List != nil:
		result := []any{}
		for _, value := range *v.List {
			result = append(result, exportValue(value))
		}
		return result
	case v.Str != nil:
		return strValue(v)
	case v.Number != nil:
		return *v.Number
	case v.Null != nil:
		return nil
	case v.Assignment != nil:
		return v.Assignment
	case v.Object != nil:
		return v.Object
	}
	return errors.New(fmt.Sprintf("unknown value type: %#v", v))
}

// FunctionCalls will return all function calls that match the given name, regardless of where they occur.
func (b *BuildScript) FunctionCalls(name string) []*FuncCall {
	result := []*FuncCall{}
	for _, f := range b.raw.FuncCalls() {
		if f.Name == name {
			result = append(result, &FuncCall{f})
		}
	}
	return result
}
