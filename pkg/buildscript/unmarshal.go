package buildscript

import (
	"errors"
	"regexp"
	"time"

	"github.com/alecthomas/participle/v2"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
)

const atTimeKey = "at_time"

var ErrOutdatedAtTime = errs.New("outdated at_time on top")

var checkoutInfoPairRegex = regexp.MustCompile(`(\w+)\s*:\s*([^\n]+)`)

// Unmarshal returns a structured form of the given AScript (on-disk format).
func Unmarshal(data []byte) (*BuildScript, error) {
	parser, err := participle.Build[rawBuildScript](participle.Unquote())
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

	// If 'at_time' is among the list of assignments, this is an outdated build script, so error out.
	for _, assignment := range raw.Assignments {
		if assignment.Key != atTimeKey {
			continue
		}
		return nil, ErrOutdatedAtTime
	}

	if raw.Info != nil {
		for _, matches := range checkoutInfoPairRegex.FindAllStringSubmatch(*raw.Info, -1) {
			key, value := matches[1], matches[2]
			switch key {
			case "Project":
				raw.CheckoutInfo.Project = value
			case "Time":
				atTime, err := strfmt.ParseDateTime(value)
				if err != nil {
					return nil, errs.Wrap(err, "Invalid timestamp: %s", value)
				}
				raw.CheckoutInfo.AtTime = time.Time(atTime)
			}
		}
	}

	return &BuildScript{raw}, nil
}
