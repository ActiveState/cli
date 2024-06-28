package ascript

import (
	"errors"
	"time"

	"github.com/alecthomas/participle/v2"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

const AtTimeKey = "at_time"

// Unmarshal returns a structured form of the given AScript (on-disk format).
func Unmarshal(data []byte) (*AScript, error) {
	parser, err := participle.Build[AScript]()
	if err != nil {
		return nil, errs.Wrap(err, "Could not create parser for build script")
	}

	script, err := parser.ParseBytes(constants.BuildScriptFileName, data)
	if err != nil {
		var parseError participle.Error
		if errors.As(err, &parseError) {
			return nil, locale.WrapExternalError(err, "err_parse_buildscript_bytes", "Could not parse build script: {{.V0}}: {{.V1}}", parseError.Position().String(), parseError.Message())
		}
		return nil, locale.WrapError(err, "err_parse_buildscript_bytes", "Could not parse build script: {{.V0}}", err.Error())
	}

	// Extract 'at_time' value from the list of assignments, if it exists.
	for i, assignment := range script.Assignments {
		key := assignment.Key
		value := assignment.Value
		if key != AtTimeKey {
			continue
		}
		script.Assignments = append(script.Assignments[:i], script.Assignments[i+1:]...)
		if value.Str == nil {
			break
		}
		atTime, err := strfmt.ParseDateTime(StrValue(value))
		if err != nil {
			return nil, errs.Wrap(err, "Invalid timestamp: %s", StrValue(value))
		}
		script.AtTime = ptr.To(time.Time(atTime))
		break
	}

	return script, nil
}
