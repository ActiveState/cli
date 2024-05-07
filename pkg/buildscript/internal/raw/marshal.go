package raw

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/buildscript/internal/buildexpression"
	"github.com/alecthomas/participle/v2"
	"github.com/go-openapi/strfmt"
)

// Marshal converts our Raw structure into a the ascript format
func (r *Raw) Marshal() ([]byte, error) {
	be, err := r.MarshalBuildExpression()
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal build expression")
	}

	expr, err := buildexpression.Unmarshal(be)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build expression")
	}

	return []byte(marshalAscriptFromBE(expr, r.AtTime)), nil
}

// MarshalBuildExpression converts our Raw structure into a build expression structure
func (r *Raw) MarshalBuildExpression() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// Unmarshal converts our ascript format into a Raw structure
func Unmarshal(data []byte) (*Raw, error) {
	parser, err := participle.Build[Raw]()
	if err != nil {
		return nil, errs.Wrap(err, "Could not create parser for build script")
	}

	r, err := parser.ParseBytes(constants.BuildScriptFileName, data)
	if err != nil {
		var parseError participle.Error
		if errors.As(err, &parseError) {
			return nil, locale.WrapExternalError(err, "err_parse_buildscript_bytes", "Could not parse build script: {{.V0}}: {{.V1}}", parseError.Position().String(), parseError.Message())
		}
		return nil, locale.WrapError(err, "err_parse_buildscript_bytes", "Could not parse build script: {{.V0}}", err.Error())
	}

	if err := r.hydrate(); err != nil {
		return nil, errs.Wrap(err, "Could not hydrate raw build script")
	}

	return r, nil
}

// UnmarshalBuildExpression converts a build expression into our raw structure
func UnmarshalBuildExpression(expr *buildexpression.BuildExpression, atTime *time.Time) (*Raw, error) {
	return Unmarshal([]byte(marshalAscriptFromBE(expr, atTime)))
}

func marshalAscriptFromBE(expr *buildexpression.BuildExpression, atTime *time.Time) string {
	buf := strings.Builder{}

	if atTime != nil {
		buf.WriteString(assignmentString(&buildexpression.Var{
			Name:  buildexpression.AtTimeKey,
			Value: &buildexpression.Value{Str: ptr.To(atTime.Format(strfmt.RFC3339Millis))},
		}))
		buf.WriteString("\n")
	}

	for _, assignment := range expr.Let.Assignments {
		if assignment.Name == buildexpression.RequirementsKey {
			assignment = transformRequirements(assignment)
		}
		buf.WriteString(assignmentString(assignment))
		buf.WriteString("\n")
	}

	buf.WriteString("\n")
	buf.WriteString("main = ")
	switch {
	case expr.Let.In.FuncCall != nil:
		buf.WriteString(apString(expr.Let.In.FuncCall))
	case expr.Let.In.Name != nil:
		buf.WriteString(*expr.Let.In.Name)
	}

	return buf.String()
}
