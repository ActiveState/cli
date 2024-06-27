package raw

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

const atTimeKey = "at_time"

// Unmarshal returns a structured form of the given AScript (on-disk format).
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

	// Extract 'at_time' value from the list of assignments, if it exists.
	for i, assignment := range r.Assignments {
		key := assignment.Key
		value := assignment.Value
		if key != atTimeKey {
			continue
		}
		r.Assignments = append(r.Assignments[:i], r.Assignments[i+1:]...)
		if value.Str == nil {
			break
		}
		atTime, err := strfmt.ParseDateTime(strValue(value))
		if err != nil {
			return nil, errs.Wrap(err, "Invalid timestamp: %s", strValue(value))
		}
		r.AtTime = ptr.To(time.Time(atTime))
		break
	}

	return r, nil
}
