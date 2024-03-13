package buildscript

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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
	expr        *buildexpression.BuildExpression
	atTime      *strfmt.DateTime
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

func NewScript(data []byte) (*Script, error) {
	parser, err := participle.Build[Script]()
	if err != nil {
		return nil, errs.Wrap(err, "Could not create parser for build script")
	}

	script, err := parser.ParseBytes(constants.BuildScriptFileName, data)
	if err != nil {
		parseErrors := errs.Unpack(err)
		if len(parseErrors) > 0 {
			return nil, locale.NewInputError("err_parse_buildscript_bytes", "Could not parse build script: {{.V0}}", parseErrors[len(parseErrors)-1].Error())
		}
		return nil, errs.Wrap(err, "Could not parse build script")
	}

	// Construct the equivalent buildexpression.
	bytes, err := json.Marshal(script)
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal build script to build expression")
	}
	expr, err := buildexpression.New(bytes)
	if err != nil {
		return nil, errs.Wrap(err, "Could not construct build expression")
	}
	script.expr = expr

	return script, nil
}

func NewScriptFromBuildExpression(expr *buildexpression.BuildExpression) (*Script, error) {
	// Copy incoming build expression to keep any modifications local.
	var err error
	expr, err = expr.Copy()
	if err != nil {
		return nil, errs.Wrap(err, "Could not copy build expression")
	}

	atTime, err := expr.SetDefaultTimestamp()
	if err != nil {
		return nil, errs.Wrap(err, "Could not set default timestamp in build expression")
	}

	return &Script{expr: expr, atTime: atTime}, nil
}

func indent(s string) string {
	return fmt.Sprintf("\t%s", strings.ReplaceAll(s, "\n", "\n\t"))
}

func (s *Script) String() string {
	buf := strings.Builder{}

	if s.atTime != nil {
		buf.WriteString(assignmentString(&buildexpression.Var{
			Name:  buildexpression.AtTimeKey,
			Value: &buildexpression.Value{Str: ptr.To(s.atTime.String())},
		}))
		buf.WriteString("\n")
	}

	for _, assignment := range s.expr.Let.Assignments {
		if assignment.Name == buildexpression.RequirementsKey {
			assignment = transformRequirements(assignment)
		}
		buf.WriteString(assignmentString(assignment))
		buf.WriteString("\n")
	}

	buf.WriteString("\n")
	buf.WriteString("main = ")
	switch {
	case s.expr.Let.In.FuncCall != nil:
		buf.WriteString(apString(s.expr.Let.In.FuncCall))
	case s.expr.Let.In.Name != nil:
		buf.WriteString(*s.expr.Let.In.Name)
	}

	return buf.String()
}

// BuildExpression returns a copy of this script's underlying buildexpression for use in a
// buildplanner context.
// For example, the solve node's "at_time" will not contain a variable reference if this script
// has a top-level assignment for it.
func (s *Script) BuildExpression() (*buildexpression.BuildExpression, error) {
	expr := s.expr

	if s.atTime != nil {
		var err error
		expr, err = expr.Copy()
		if err != nil {
			return nil, errs.Wrap(err, "Failed to copy buildexpression")
		}
		err = expr.MaybeUpdateTimestamp(*s.atTime)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to possibly update %s", buildexpression.AtTimeKey)
		}
	}

	return expr, nil
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
//	Req(name = "<namespace>/<name>", version = <op>("<version>"))
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
//	And(<op1>("<version1>"), <op2>("<version2>"))
func transformVersion(requirements *buildexpression.Var) *buildexpression.Ap {
	var aps []*buildexpression.Ap
	for _, constraint := range *requirements.Value.List {
		ap := &buildexpression.Ap{}
		for _, o := range *constraint.Object {
			switch o.Name {
			case buildexpression.RequirementVersionKey:
				ap.Arguments = []*buildexpression.Value{{Str: o.Value.Str}}
			case buildexpression.RequirementComparatorKey:
				ap.Name = strings.Title(*o.Value.Str)
			}
		}
		aps = append(aps, ap)
	}

	if len(aps) == 1 {
		return aps[0] // e.g. Eq("1.0")
	}

	args := make([]*buildexpression.Value, len(aps))
	for i, ap := range aps {
		args[i] = &buildexpression.Value{Ap: ap}
	}
	return &buildexpression.Ap{Name: andFuncName, Arguments: args} // e.g. And(Gt("1.0"), Lt("3.0"))
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

	return fmt.Sprintf("[\n]") // participle does not create v.List if it's empty
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

func (s *Script) EqualsBuildExpressionBytes(exprBytes []byte) bool {
	expr, err := buildexpression.New(exprBytes)
	if err != nil {
		multilog.Error("Unable to create buildexpression from incoming JSON: %v", err)
		return false
	}
	return s.EqualsBuildExpression(expr)
}

func (s *Script) EqualsBuildExpression(expr *buildexpression.BuildExpression) bool {
	thisExpr, err := s.BuildExpression()
	if err != nil {
		multilog.Error("Unable to compute buildexpression from this buildscript: %v", err)
		return false
	}
	myJson, err := json.Marshal(thisExpr)
	if err != nil {
		multilog.Error("Unable to marshal this buildscript to JSON: %v", err)
		return false
	}
	otherJson, err := json.Marshal(expr)
	if err != nil {
		multilog.Error("Unable to marshal other buildscript to JSON: %v", err)
		return false
	}
	return string(myJson) == string(otherJson)
}
