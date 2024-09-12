package buildscript_runbit

import (
	"errors"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/checkoutinfo"
)

func CommitID(path string, cfg configurer) (strfmt.UUID, error) {
	script, err := buildscript.New()
	if err != nil {
		return "", errs.Wrap(err, "Could not create build script")
	}

	project, err := checkoutinfo.GetProject(path)
	if err != nil {
		return "", errs.Wrap(err, "Could not get project")
	}
	script.SetProjectURL(project) // script.CommitID() is extracted from project URL

	if cfg.GetBool(constants.OptinBuildscriptsConfig) {
		if script2, err := ScriptFromProject(path); err == nil {
			script = script2
		} else if !errors.Is(err, ErrBuildscriptNotExist) {
			return "", errs.Wrap(err, "Could not get build script")
		}
		// ErrBuildscriptNotExist will fall back on activestate.yaml
	}

	return script.CommitID()
}
