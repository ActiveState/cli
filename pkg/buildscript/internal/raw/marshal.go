package raw

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/buildscript/internal/buildexpression"
)

const mainKey = "main"

// Marshal returns this structure in AScript, suitable for writing to disk.
func (r *Raw) Marshal() ([]byte, error) {
	be, err := r.MarshalBuildExpression()
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal build expression")
	}

	expr, err := buildexpression.Unmarshal(be)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build expression")
	}

	return marshalFromBuildExpression(expr, r.AtTime), nil
}

// MarshalBuildExpression converts our Raw structure into a build expression structure
func (r *Raw) MarshalBuildExpression() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// marshalFromBuildExpression is a bit special in that it is sort of an unmarshaller and a marshaller at the same time.
// It takes a build expression and directly translates it into the string representation of a build script.
// We should update this so that this instead translates the build expression directly to the in-memory representation
// of a buildscript (ie. the Raw structure). But that is a large refactor in and of itself that'll follow later.
// For now we can use this to convert a build expression to a buildscript with an extra hop where we have to unmarshal
// the resulting buildscript string.
func marshalFromBuildExpression(expr *buildexpression.BuildExpression, atTime *time.Time) []byte {
	buf := strings.Builder{}

	if atTime != nil {
		buf.WriteString(assignmentString(&buildexpression.Var{
			Name:  buildexpression.AtTimeKey,
			Value: &buildexpression.Value{Str: ptr.To(atTime.Format(strfmt.RFC3339Millis))},
		}))
		buf.WriteString("\n")
	}

	for _, assignment := range expr.Let.Assignments {
		if assignment.Name == buildexpression.RequirementsKey && isLegacyRequirementsList(assignment) {
			assignment = transformRequirements(assignment)
		}
		buf.WriteString(assignmentString(assignment))
		buf.WriteString("\n")
	}

	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf("%s = ", mainKey))
	switch {
	case expr.Let.In.FuncCall != nil:
		buf.WriteString(apString(expr.Let.In.FuncCall))
	case expr.Let.In.Name != nil:
		buf.WriteString(*expr.Let.In.Name)
	}

	return []byte(buf.String())
}

func assignmentString(a *buildexpression.Var) string {
	if a.Name == buildexpression.RequirementsKey && isLegacyRequirementsList(a) {
		a = transformRequirements(a)
	}
	return fmt.Sprintf("%s = %s", a.Name, valueString(a.Value))
}

func indent(s string) string {
	return fmt.Sprintf("\t%s", strings.ReplaceAll(s, "\n", "\n\t"))
}

func valueString(v *buildexpression.Value) string {
	switch {
	case v.Ap != nil:
		return apString(v.Ap)

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
		if strings.HasPrefix(*v.Str, "$") { // variable reference
			return strings.TrimLeft(*v.Str, "$")
		}
		return strconv.Quote(*v.Str)

	case v.Float != nil:
		return strconv.FormatFloat(*v.Float, 'G', -1, 64) // 64-bit float with minimum digits on display

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

// inlineFunctions contains buildscript function names whose arguments should all be written on a
// single line. By default, function arguments are written one per line.
var inlineFunctions = []string{
	reqFuncName,
	eqFuncName, neFuncName,
	gtFuncName, gteFuncName,
	ltFuncName, lteFuncName,
	andFuncName,
}

func apString(f *buildexpression.Ap) string {
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
