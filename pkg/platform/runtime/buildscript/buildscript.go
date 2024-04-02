package buildscript

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/alecthomas/participle/v2"
)

// Script's tagged fields will be initially filled in by Participle.
// expr will be constructed later and is this script's buildexpression. We keep a copy of the build
// expression here with any changes that have been applied before either writing it to disk or
// submitting it to the build planner. It's easier to operate on build expressions directly than to
// modify or manually populate the Participle-produced fields and re-generate a build expression.
type Script struct {
	Assignments []*Assignment `parser:"@@+"`
	AtTime      *strfmt.DateTime
	Expr        *buildexpression.BuildExpression
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
	Ident      *string        `parser:"| @Ident"`                    // only in FuncCall or Assignment
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

var (
	reqFuncName = "Req"
	eqFuncName  = "Eq"
	neFuncName  = "Ne"
	gtFuncName  = "Gt"
	gteFuncName = "Gte"
	ltFuncName  = "Lt"
	lteFuncName = "Lte"
	andFuncName = "And"
)

func New(data []byte) (*Script, error) {
	parser, err := participle.Build[Script]()
	if err != nil {
		return nil, errs.Wrap(err, "Could not create parser for build script")
	}

	script, err := parser.ParseBytes(constants.BuildScriptFileName, data)
	if err != nil {
		var parseError participle.Error
		if errors.As(err, &parseError) {
			return nil, locale.WrapInputError(err, "err_parse_buildscript_bytes", "Could not parse build script: {{.V0}}: {{.V1}}", parseError.Position().String(), parseError.Message())
		}
		return nil, locale.WrapError(err, "err_parse_buildscript_bytes", "Could not parse build script: {{.V0}}", err.Error())
	}

	// Construct the equivalent buildexpression.
	bytes, err := json.Marshal(script)
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal build script to build expression")
	}

	expr, err := buildexpression.New(bytes)
	if err != nil {
		return nil, locale.WrapError(err, "err_parse_buildscript_bytes", "Could not construct build expression: {{.V0}}", errs.JoinMessage(err))
	}
	script.Expr = expr

	return script, nil
}

func NewFromBuildExpression(atTime *strfmt.DateTime, expr *buildexpression.BuildExpression) (*Script, error) {
	// Copy incoming build expression to keep any modifications local.
	var err error
	expr, err = expr.Copy()
	if err != nil {
		return nil, errs.Wrap(err, "Could not copy build expression")
	}

	// Update old expressions that bake in at_time as a timestamp instead of as a variable.
	err = expr.MaybeSetDefaultTimestamp(atTime)
	if err != nil {
		return nil, errs.Wrap(err, "Could not set default timestamp")
	}

	return &Script{AtTime: atTime, Expr: expr}, nil
}

func indent(s string) string {
	return fmt.Sprintf("\t%s", strings.ReplaceAll(s, "\n", "\n\t"))
}

func (s *Script) String() string {
	buf := strings.Builder{}

	if s.AtTime != nil {
		buf.WriteString(assignmentString(&buildexpression.Var{
			Name:  buildexpression.AtTimeKey,
			Value: &buildexpression.Value{Str: ptr.To(s.AtTime.String())},
		}))
		buf.WriteString("\n")
	}

	for _, assignment := range s.Expr.Let.Assignments {
		if assignment.Name == buildexpression.RequirementsKey {
			assignment = transformRequirements(assignment)
		}
		buf.WriteString(assignmentString(assignment))
		buf.WriteString("\n")
	}

	buf.WriteString("\n")
	buf.WriteString("main = ")
	switch {
	case s.Expr.Let.In.FuncCall != nil:
		buf.WriteString(apString(s.Expr.Let.In.FuncCall))
	case s.Expr.Let.In.Name != nil:
		buf.WriteString(*s.Expr.Let.In.Name)
	}

	return buf.String()
}

// transformRequirements transforms a buildexpression list of requirements in object form into a
// list of requirements in function-call form, which is how requirements are represented in
// buildscripts.
// This is to avoid custom marshaling code and reuse existing marshaling code.
func transformRequirements(reqs *buildexpression.Var) *buildexpression.Var {
	newReqs := &buildexpression.Var{
		Name: buildexpression.RequirementsKey,
		Value: &buildexpression.Value{
			List: &[]*buildexpression.Value{},
		},
	}

	for _, req := range *reqs.Value.List {
		*newReqs.Value.List = append(*newReqs.Value.List, transformRequirement(req))
	}

	return newReqs
}

// transformRequirement transforms a buildexpression requirement in object form into a requirement
// in function-call form.
// For example, transform something like
//
//	{"name": "<name>", "namespace": "<namespace>",
//		"version_requirements": [{"comparator": "<op>", "version": "<version>"}]}
//
// into something like
//
//	Req(name = "<namespace>/<name>", version = <op>(value = "<version>"))
func transformRequirement(req *buildexpression.Value) *buildexpression.Value {
	newReq := &buildexpression.Value{
		Ap: &buildexpression.Ap{
			Name:      reqFuncName,
			Arguments: []*buildexpression.Value{},
		},
	}

	// Extract namespace, name, and version from requirement object.
	name := ""
	var version *buildexpression.Ap
	for _, arg := range *req.Object {
		switch arg.Name {
		case buildexpression.RequirementNameKey:
			name += *arg.Value.Str

		case buildexpression.RequirementNamespaceKey:
			name = fmt.Sprintf("%s/%s", *arg.Value.Str, name)

		case buildexpression.RequirementVersionRequirementsKey:
			version = transformVersion(arg)
		}
	}

	// Add the arguments to the function transformation.
	newReq.Ap.Arguments = append(newReq.Ap.Arguments, &buildexpression.Value{
		Assignment: &buildexpression.Var{
			Name:  buildexpression.RequirementNameKey,
			Value: &buildexpression.Value{Str: ptr.To(name)},
		},
	})
	if version != nil {
		newReq.Ap.Arguments = append(newReq.Ap.Arguments, &buildexpression.Value{
			Assignment: &buildexpression.Var{
				Name:  buildexpression.RequirementVersionKey,
				Value: &buildexpression.Value{Ap: version},
			},
		})
	}

	return newReq
}

// transformVersion transforms a buildexpression version_requirements list in object form into
// function-call form.
// For example, transform something like
//
//	[{"comparator": "<op1>", "version": "<version1>"}, {"comparator": "<op2>", "version": "<version2>"}]
//
// into something like
//
//	And(<op1>(value = "<version1>"), <op2>(value = "<version2>"))
func transformVersion(requirements *buildexpression.Var) *buildexpression.Ap {
	var aps []*buildexpression.Ap
	for _, constraint := range *requirements.Value.List {
		ap := &buildexpression.Ap{}
		for _, o := range *constraint.Object {
			switch o.Name {
			case buildexpression.RequirementVersionKey:
				ap.Arguments = []*buildexpression.Value{{
					Assignment: &buildexpression.Var{Name: "value", Value: &buildexpression.Value{Str: o.Value.Str}},
				}}
			case buildexpression.RequirementComparatorKey:
				ap.Name = cases.Title(language.English).String(*o.Value.Str)
			}
		}
		aps = append(aps, ap)
	}

	if len(aps) == 1 {
		return aps[0] // e.g. Eq(value = "1.0")
	}

	// e.g. And(left = Gt(value = "1.0"), right = Lt(value = "3.0"))
	// Iterate backwards over the requirements array and construct a binary tree of 'And()' functions.
	// For example, given [Gt(value = "1.0"), Ne(value = "2.0"), Lt(value = "3.0")], produce:
	//   And(left = Gt(value = "1.0"), right = And(left = Ne(value = "2.0"), right = Lt(value = "3.0")))
	var ap *buildexpression.Ap
	for i := len(aps) - 2; i >= 0; i-- {
		right := &buildexpression.Value{Ap: aps[i+1]}
		if ap != nil {
			right = &buildexpression.Value{Ap: ap}
		}
		args := []*buildexpression.Value{
			{Assignment: &buildexpression.Var{Name: "left", Value: &buildexpression.Value{Ap: aps[i]}}},
			{Assignment: &buildexpression.Var{Name: "right", Value: right}},
		}
		ap = &buildexpression.Ap{Name: andFuncName, Arguments: args}
	}
	return ap
}

func assignmentString(a *buildexpression.Var) string {
	if a.Name == buildexpression.RequirementsKey {
		a = transformRequirements(a)
	}
	return fmt.Sprintf("%s = %s", a.Name, valueString(a.Value))
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

func (s *Script) Equals(other *Script) bool {
	// Compare top-level at_time.
	switch {
	case s.AtTime != nil && other.AtTime != nil && s.AtTime.String() != other.AtTime.String():
		return false
	case (s.AtTime == nil) != (other.AtTime == nil):
		return false
	}

	// Compare buildexpression JSON.
	myJson, err := json.Marshal(s.Expr)
	if err != nil {
		multilog.Error("Unable to marshal this buildscript to JSON: %v", err)
		return false
	}
	otherJson, err := json.Marshal(other.Expr)
	if err != nil {
		multilog.Error("Unable to marshal other buildscript to JSON: %v", err)
		return false
	}
	return string(myJson) == string(otherJson)
}
