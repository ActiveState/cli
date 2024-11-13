package buildscript

import (
	"errors"
	"strings"
	"time"

	"github.com/alecthomas/participle/v2"
	"github.com/go-openapi/strfmt"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
)

const atTimeKey = "at_time"

var ErrOutdatedAtTime = errs.New("outdated at_time on top")

type checkoutInfo struct {
	Project string `yaml:"Project"`
	Time    string `yaml:"Time"`
}

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

	// Verify there are no duplicate key assignments.
	// This is primarily to catch duplicate solve nodes for a given target.
	seen := make(map[string]bool)
	for _, assignment := range raw.Assignments {
		if _, exists := seen[assignment.Key]; exists {
			return nil, locale.NewInputError(locale.Tl("err_buildscript_duplicate_keys", "Build script has duplicate '{{.V0}}' assignments", assignment.Key))
		}
		seen[assignment.Key] = true
	}

	var project string
	var atTime *time.Time
	if raw.Info != nil {
		info := checkoutInfo{}

		err := yaml.Unmarshal([]byte(strings.Trim(*raw.Info, "`\n")), &info)
		if err != nil {
			return nil, locale.NewInputError(
				"err_buildscript_checkoutinfo",
				"Could not parse checkout information in the buildscript. The parser produced the following error: {{.V0}}", err.Error())
		}

		project = info.Project

		atTimeVal, err := time.Parse(time.RFC3339, info.Time)
		if err != nil {
			// Older buildscripts used microsecond specificity
			atDateTime, err := strfmt.ParseDateTime(info.Time)
			if err != nil {
				return nil, errs.Wrap(err, "Invalid timestamp: %s", info.Time)
			}
			atTimeVal = time.Time(atDateTime)
		}
		atTime = &atTimeVal
	}

	return &BuildScript{raw, project, atTime}, nil
}
