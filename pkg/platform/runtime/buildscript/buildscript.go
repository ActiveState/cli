package buildscript

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/alecthomas/participle/v2"
)

type Script struct {
	Let *Let `parser:"'let' ':' @@"`
	In  *In  `parser:"'in' ':' @@"`
}

type Let struct {
	Assignments []*Assignment `parser:"@@+"`
}

type Assignment struct {
	Key   string `parser:"@Ident '='"`
	Value *Value `parser:"@@"`
}

type Value struct {
	FuncCall *FuncCall `parser:"@@"`
	List     *[]*Value `parser:"| '[' (@@ (',' @@)* ','?)? ']'"`
	Str      *string   `parser:"| @String"`
	Number   *float64  `parser:"| (@Float | @Int)"`
	Null     *Null     `parser:"| @@"`

	Assignment *Assignment    `parser:"| @@"`                        // only in FuncCall
	Object     *[]*Assignment `parser:"| '{' @@ (',' @@)* ','? '}'"` // only in List
	Ident      *string        `parser:"| @Ident"`                    // only in FuncCall
}

type Null struct {
	Null string `parser:"'null'"`
}

type FuncCall struct {
	Name      string   `parser:"@Ident"`
	Arguments []*Value `parser:"'(' @@ (',' @@)* ','? ')'"`
}

type In struct {
	FuncCall *FuncCall `parser:"@@"`
	Name     *string   `parser:"| @Ident"`
}

func NewScript(data []byte) (*Script, error) {
	parser, err := participle.Build[Script]()
	if err != nil {
		return nil, errs.Wrap(err, "Could not create parser for build script")
	}

	script, err := parser.ParseBytes(constants.BuildScriptFileName, data)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse build script")
	}

	return script, nil
}

func indent(s string) string {
	return fmt.Sprintf("\t%s", strings.ReplaceAll(s, "\n", "\n\t"))
}

func (s *Script) String() string {
	buf := strings.Builder{}
	buf.WriteString("let:\n")
	for _, assignment := range s.Let.Assignments {
		buf.WriteString(indent(assignment.String()))
	}
	buf.WriteString("\n\n")
	buf.WriteString("in:\n")
	switch {
	case s.In.FuncCall != nil:
		buf.WriteString(indent(s.In.FuncCall.String()))
	case s.In.Name != nil:
		buf.WriteString(indent(*s.In.Name))
	}
	return buf.String()
}

func (a *Assignment) String() string {
	return fmt.Sprintf("%s = %s", a.Key, a.Value.String())
}

func (v *Value) String() string {
	switch {
	case v.FuncCall != nil:
		return v.FuncCall.String()

	case v.List != nil:
		buf := bytes.Buffer{}
		buf.WriteString("[\n")
		for i, item := range *v.List {
			buf.WriteString(indent(item.String()))
			if i+1 < len(*v.List) {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
		}
		buf.WriteString("]")
		return buf.String()

	case v.Str != nil:
		return *v.Str

	case v.Number != nil:
		return strconv.FormatFloat(*v.Number, 'G', -1, 64)

	case v.Null != nil:
		return "null"

	case v.Assignment != nil:
		return v.Assignment.String()

	case v.Object != nil:
		buf := bytes.Buffer{}
		buf.WriteString("{\n")
		for i, pair := range *v.Object {
			buf.WriteString(indent(pair.String()))
			if i+1 < len(*v.Object) {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
		}
		buf.WriteString("}")
		return buf.String()

	case v.Ident != nil:
		return *v.Ident
	}

	return fmt.Sprintf("[\n]") // participle does not create v.List if it's empty
}

func (f *FuncCall) String() string {
	buf := bytes.Buffer{}
	buf.WriteString(fmt.Sprintf("%s(\n", f.Name))
	for i, argument := range f.Arguments {
		buf.WriteString(indent(argument.String()))
		if i+1 < len(f.Arguments) {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString(")")
	return buf.String()
}
