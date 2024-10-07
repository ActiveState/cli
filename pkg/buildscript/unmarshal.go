package buildscript

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
func Unmarshal(data []byte) (*BuildScript, error) {
	return UnmarshalWithProcessors(data, DefaultProcessors)
}

func UnmarshalWithProcessors(data []byte, processors FuncProcessorMap) (*BuildScript, error) {
	parser, err := participle.Build[rawBuildScript]()
	if err != nil {
		return nil, errs.Wrap(err, "Could not create parser for build script")
	}

	raw, err := parser.ParseBytes(constants.BuildScriptFileName, data)
	if err != nil {
		var parseError participle.Error
		if errors.As(err, &parseError) {
			return nil, locale.WrapExternalError(err, "err_parse_buildscript_bytes", "Could not parse build script: {{.V0}}: {{.V1}}", parseError.Position().String(), parseError.Message())
		}
		return nil, locale.WrapError(err, "err_parse_buildscript_bytes", "Could not parse build script: {{.V0}}", err.Error())
	}

	// Extract 'at_time' value from the list of assignments, if it exists.
	for i, assignment := range raw.Assignments {
		key := assignment.Key
		value := assignment.Value
		if key != atTimeKey {
			continue
		}
		raw.Assignments = append(raw.Assignments[:i], raw.Assignments[i+1:]...)
		if value.Str == nil {
			break
		}
		atTime, err := strfmt.ParseDateTime(strValue(value))
		if err != nil {
			return nil, errs.Wrap(err, "Invalid timestamp: %s", strValue(value))
		}
		raw.AtTime = ptr.To(time.Time(atTime))
		break
	}

	return &BuildScript{raw, processors}, nil
}
