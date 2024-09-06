package buildscript

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
)

const (
	mainKey = "main"

	reqFuncName = "Req"
	revFuncName = "Revision"
	eqFuncName  = "Eq"
	neFuncName  = "Ne"
	gtFuncName  = "Gt"
	gteFuncName = "Gte"
	ltFuncName  = "Lt"
	lteFuncName = "Lte"
	andFuncName = "And"
)

// Marshal returns this structure in AScript, suitable for writing to disk.
func (b *BuildScript) Marshal() ([]byte, error) {
	buf := strings.Builder{}

	buf.WriteString("```\n")
	buf.WriteString("Project: " + b.raw.CheckoutInfo.Project + "\n")
	buf.WriteString("Time: " + b.raw.CheckoutInfo.AtTime.Format(strfmt.RFC3339Millis) + "\n")
	buf.WriteString("```\n\n")

	var main *Assignment
	for _, assignment := range b.raw.Assignments {
		if assignment.Key == mainKey {
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

func indentByTab(s string) string {
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
			buf.WriteString(indentByTab(valueString(item)))
			if i+1 < len(*v.List) {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
		}
		buf.WriteString("]")
		return buf.String()

	case v.Str != nil:
		return strconv.Quote(*v.Str)

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
			buf.WriteString(indentByTab(assignmentString(pair)))
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
	reqFuncName,
	eqFuncName, neFuncName,
	gtFuncName, gteFuncName,
	ltFuncName, lteFuncName,
	andFuncName,
}

func funcCallString(f *FuncCall) string {
	var (
		newline = "\n"
		comma   = ","
		indent  = indentByTab
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

	buf.WriteString(argsToString(f.Arguments, newline, comma, indent))

	buf.WriteString(")")
	return buf.String()
}

func argsToString(args []*Value, newline, comma string, indent func(string) string) string {
	buf := bytes.Buffer{}
	for i, argument := range args {
		buf.WriteString(indent(valueString(argument)))

		if i+1 < len(args) {
			buf.WriteString(comma)
		}

		buf.WriteString(newline)
	}
	return buf.String()
}
