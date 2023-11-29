package buildscript

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/alecthomas/participle/v2"
)

// Version is the current buildscript version.
const CurrentVersion int = 1

// Script's tagged fields will be initially filled in by Participle.
// Expr will be constructed later and is this script's buildexpression. We keep a copy of the build
// expression here with any changes that have been applied before either writing it to disk or
// submitting it to the build planner. It's easier to operate on build expressions directly than to
// modify or manually populate the Participle-produced fields and re-generate a build expression.
type Script struct {
	Let     *Let `parser:"'let' ':' @@"`
	In      *In  `parser:"'in' ':' @@"`
	Expr    *buildexpression.BuildExpression
	Version int `parser:"'version' ':' @Int"`
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

	// Construct the equivalent buildexpression.
	bytes, err := json.Marshal(script)
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal build script to build expression")
	}
	expr, err := buildexpression.New(bytes)
	if err != nil {
		return nil, errs.Wrap(err, "Could not construct build expression")
	}
	script.Expr = expr

	// Migrate from older buildscript versions if necessary.
	if script.Version < CurrentVersion {
		err := migrateOldBuildScript(script)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to migrate old build script")
		}
	}

	return script, nil
}

func migrateOldBuildScript(script *Script) error {
	return nil // Noop for now until we need to migrate
}

func NewScriptFromBuildExpression(expr *buildexpression.BuildExpression) (*Script, error) {
	return &Script{Expr: expr, Version: CurrentVersion}, nil
}

func indent(s string) string {
	return fmt.Sprintf("\t%s", strings.ReplaceAll(s, "\n", "\n\t"))
}

func (s *Script) String() string {
	buf := strings.Builder{}
	buf.WriteString("let:\n")
	for _, assignment := range s.Expr.Let.Assignments {
		buf.WriteString(indent(assignmentString(assignment)))
		buf.WriteString("\n")
	}
	buf.WriteString("\n")
	buf.WriteString("in:\n")
	switch {
	case s.Expr.Let.In.FuncCall != nil:
		buf.WriteString(indent(apString(s.Expr.Let.In.FuncCall)))
	case s.Expr.Let.In.Name != nil:
		buf.WriteString(indent(*s.Expr.Let.In.Name))
	}
	buf.WriteString("\n\n")
	buf.WriteString("version: ")
	buf.WriteString(strconv.Itoa(s.Version))
	return buf.String()
}

func assignmentString(a *buildexpression.Var) string {
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

func apString(f *buildexpression.Ap) string {
	buf := bytes.Buffer{}
	buf.WriteString(fmt.Sprintf("%s(\n", f.Name))
	for i, argument := range f.Arguments {
		buf.WriteString(indent(valueString(argument)))
		if i+1 < len(f.Arguments) {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
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
	myJson, err := json.Marshal(s.Expr)
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
