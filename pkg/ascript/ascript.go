package ascript

import (
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

// Tagged fields will be filled in by Participle.
type AScript struct {
	Assignments []*Assignment `parser:"@@+"`

	AtTime *time.Time // set after initial read
}

type Assignment struct {
	Key   string `parser:"@Ident '='"`
	Value *Value `parser:"@@"`
}

type Value struct {
	FuncCall *FuncCall `parser:"@@"`
	List     *[]*Value `parser:"| '[' (@@ (',' @@)* ','?)? ']'"`
	Str      *string   `parser:"| @String"` // note: this value is ALWAYS quoted
	Number   *float64  `parser:"| (@Float | @Int)"`
	Null     *Null     `parser:"| @@"`

	Assignment *Assignment    `parser:"| @@"`                        // only in FuncCall
	Object     *[]*Assignment `parser:"| '{' @@ (',' @@)* ','? '}'"` // only in List
	Ident      *string        `parser:"| @Ident"`                    // only in FuncCall or Assignment
}

type Null struct {
	Null string `parser:"'null'"`
}

type FuncCall struct {
	Name      string   `parser:"@Ident"`
	Arguments []*Value `parser:"'(' @@ (',' @@)* ','? ')'"`
}

// NewString is a convenience function for constructing a string Value from an unquoted string.
// Use this instead of &Value{Str: ptr.To(strconv.Quote(s))}
func NewString(s string) *Value {
	return &Value{Str: ptr.To(strconv.Quote(s))}
}

// StrValue is a convenience function for retrieving an unquoted string from Value.
// Use this instead of strings.Trim(*v.Str, `"`)
func StrValue(v *Value) string {
	return strings.Trim(*v.Str, `"`)
}
