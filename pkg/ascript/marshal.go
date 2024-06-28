package ascript

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
)

const (
	MainKey = "main"

	ReqFuncName = "Req"
	EqFuncName  = "Eq"
	NeFuncName  = "Ne"
	GtFuncName  = "Gt"
	GteFuncName = "Gte"
	LtFuncName  = "Lt"
	LteFuncName = "Lte"
	AndFuncName = "And"
)

// Marshal returns this structure in AScript, suitable for writing to disk.
func (a *AScript) Marshal() ([]byte, error) {
	buf := strings.Builder{}

	if a.AtTime != nil {
		buf.WriteString(assignmentString(
			&Assignment{AtTimeKey, NewString(a.AtTime.Format(strfmt.RFC3339Millis))}))
		buf.WriteString("\n")
	}

	var main *Assignment
	for _, assignment := range a.Assignments {
		if assignment.Key == MainKey {
			main = assignment
			continue // write at the end
		}
		buf.WriteString(assignmentString(assignment))
		buf.WriteString("\n")
	}

	buf.WriteString("\n")
	buf.WriteString(assignmentString(main))

	return []byte(buf.String()), nil
}

func assignmentString(a *Assignment) string {
	return fmt.Sprintf("%s = %s", a.Key, valueString(a.Value))
}

func indent(s string) string {
	return fmt.Sprintf("\t%s", strings.ReplaceAll(s, "\n", "\n\t"))
}

func valueString(v *Value) string {
	switch {
	case v.FuncCall != nil:
		return funcCallString(v.FuncCall)

	case v.List != nil:
		buf := bytes.Buffer{}
		buf.WriteString("[\n")
		for i, item := range *v.List {
			buf.WriteString(indent(valueString(item)))
			if i+1 < len(*v.List) {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
		}
		buf.WriteString("]")
		return buf.String()

	case v.Str != nil:
		return *v.Str // keep quoted

	case v.Number != nil:
		return strconv.FormatFloat(*v.Number, 'G', -1, 64) // 64-bit float with minimum digits on display

	case v.Null != nil:
		return "null"

	case v.Assignment != nil:
		return assignmentString(v.Assignment)

	case v.Object != nil:
		buf := bytes.Buffer{}
		buf.WriteString("{\n")
		for i, pair := range *v.Object {
			buf.WriteString(indent(assignmentString(pair)))
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

	return "[\n]" // participle does not create v.List if it's empty
}

// inlineFunctions contains build script function names whose arguments should all be written on a
// single line. By default, function arguments are written one per line.
var inlineFunctions = []string{
	ReqFuncName,
	EqFuncName, NeFuncName,
	GtFuncName, GteFuncName,
	LtFuncName, LteFuncName,
	AndFuncName,
}

func funcCallString(f *FuncCall) string {
	var (
		newline = "\n"
		comma   = ","
		indent  = indent
	)

	if funk.Contains(inlineFunctions, f.Name) {
		newline = ""
		comma = ", "
		indent = func(s string) string {
			return s
		}
	}

	buf := bytes.Buffer{}
	buf.WriteString(fmt.Sprintf("%s(%s", f.Name, newline))

	for i, argument := range f.Arguments {
		buf.WriteString(indent(valueString(argument)))

		if i+1 < len(f.Arguments) {
			buf.WriteString(comma)
		}

		buf.WriteString(newline)
	}

	buf.WriteString(")")
	return buf.String()
}
