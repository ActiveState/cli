package buildscript

import (
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/brunoga/deep"
)

// Tagged fields will be filled in by Participle.
type rawBuildScript struct {
	Assignments []*assignment `parser:"@@+"`

	AtTime *time.Time // set after initial read
}

// clone is meant to facilitate making modifications to functions at marshal time. The idea is that these modifications
// are only intended to be made for the purpose of marshalling, meaning we do not want to mutate the original object.
// This is an antipattern, but addressing it requires significant refactoring that we're not committing to atm.
func (r *rawBuildScript) clone() (*rawBuildScript, error) {
	return deep.Copy(r)
}

func (r *rawBuildScript) FuncCalls() []*funcCall {
	result := []*funcCall{}
	for _, a := range r.Assignments {
		result = append(result, a.Value.funcCalls()...)
	}
	return result
}

// funcCalls will return all function calls recursively under the given value.
func (v *value) funcCalls() []*funcCall {
	result := []*funcCall{}
	switch {
	case v.FuncCall != nil:
		result = append(result, v.FuncCall)
		for _, arg := range v.FuncCall.Arguments {
			result = append(result, arg.funcCalls()...)
		}
	case v.List != nil:
		for _, v := range *v.List {
			result = append(result, v.funcCalls()...)
		}
	case v.Assignment != nil:
		result = append(result, v.Assignment.Value.funcCalls()...)
	case v.Object != nil:
		for _, a := range *v.Object {
			result = append(result, a.Value.funcCalls()...)
		}
	}
	return result
}

type assignment struct {
	Key   string `parser:"@Ident '='"`
	Value *value `parser:"@@"`
}

type value struct {
	FuncCall *funcCall `parser:"@@"`
	List     *[]*value `parser:"| '[' (@@ (',' @@)* ','?)? ']'"`
	Str      *string   `parser:"| @String"` // note: this value is ALWAYS quoted
	Number   *float64  `parser:"| (@Float | @Int)"`
	Null     *null     `parser:"| @@"`

	Assignment *assignment    `parser:"| @@"`                        // only in FuncCall
	Object     *[]*assignment `parser:"| '{' @@ (',' @@)* ','? '}'"` // only in List
	Ident      *string        `parser:"| @Ident"`                    // only in FuncCall or Assignment
}

type null struct {
	Null string `parser:"'null'"`
}

type funcCall struct {
	Name      string   `parser:"@Ident"`
	Arguments []*value `parser:"'(' @@ (',' @@)* ','? ')'"`
}

// newString is a convenience function for constructing a string value from an unquoted string.
// Use this instead of &value{Str: ptr.To(strconv.Quote(s))}
func newString(s string) *value {
	return &value{Str: ptr.To(strconv.Quote(s))}
}

// strValue is a convenience function for retrieving an unquoted string from value.
// Use this instead of strings.Trim(*v.Str, `"`)
func strValue(v *value) string {
	return strings.Trim(*v.Str, `"`)
}
