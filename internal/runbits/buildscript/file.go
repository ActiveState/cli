package buildscript_runbit

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
)

// projecter is a union between project.Project and setup.Targeter
type projecter interface {
	Dir() string
	Owner() string
	Name() string
	BranchName() string
}

var ErrBuildscriptNotExist = errors.New("Build script does not exist")

func ScriptFromProject(proj projecter) (*buildscript.BuildScript, error) {
	path := filepath.Join(proj.Dir(), constants.BuildScriptFileName)
	return ScriptFromFile(path)
}

func ScriptFromFile(path string) (*buildscript.BuildScript, error) {
	data, err := fileutils.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errs.Pack(err, ErrBuildscriptNotExist)
		}
		return nil, errs.Wrap(err, "Could not read build script from file")
	}
	return buildscript.Unmarshal(data)
}

type primeable interface {
	primer.Auther
	primer.SvcModeler
}

// Initialize creates a new build script for the local project. It will overwrite an existing one so
// commands like `state reset` will work.
func Initialize(proj projecter, auth *authentication.Auth, svcm *model.SvcModel) error {
	logging.Debug("Initializing build script")
	commitId, err := localcommit.Get(proj.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get the local commit ID")
	}

	buildplanner := buildplanner.NewBuildPlannerModel(auth, svcm)
	script, err := buildplanner.GetBuildScript(commitId.String())
	if err != nil {
		return errs.Wrap(err, "Unable to get the remote build expression and time")
	}

	if url, err := projectURL(proj, commitId.String()); err == nil {
		script.SetProject(url)
	} else {
		return errs.Wrap(err, "Unable to set project")
	}
	// Note: script.SetAtTime() was done in GetBuildScript().

	scriptBytes, err := script.Marshal()
	if err != nil {
		return errs.Wrap(err, "Unable to marshal build script")
	}

	scriptPath := filepath.Join(proj.Dir(), constants.BuildScriptFileName)
	logging.Debug("Initializing build script at %s", scriptPath)
	err = fileutils.WriteFile(scriptPath, scriptBytes)
	if err != nil {
		return errs.Wrap(err, "Unable to write build script")
	}

	return nil
}

func projectURL(proj projecter, commitID string) (string, error) {
	// Note: cannot use api.GetPlatformURL() due to import cycle.
	host := constants.DefaultAPIHost
	if hostOverride := os.Getenv(constants.APIHostEnvVarName); hostOverride != "" {
		host = hostOverride
	}
	u, err := url.Parse(fmt.Sprintf("https://%s/%s/%s", host, proj.Owner(), proj.Name()))
	if err != nil {
		return "", errs.Wrap(err, "Unable to parse URL")
	}
	q := u.Query()
	q.Set("branch", proj.BranchName())
	q.Set("commitID", commitID)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func Update(proj projecter, newScript *buildscript.BuildScript) error {
	script, err := ScriptFromProject(proj)
	if err != nil {
		return errs.Wrap(err, "Could not read build script")
	}

	equals, err := script.Equals(newScript)
	if err != nil {
		return errs.Wrap(err, "Could not compare build script")
	}
	if script != nil && equals {
		return nil // no changes to write
	}

	// Update the new script's project field to match the current one, except for a new commit ID.
	commitID, err := localcommit.Get(proj.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get the local commit ID")
	}
	url, err := projectURL(proj, commitID.String())
	if err != nil {
		return errs.Wrap(err, "Could not construct project URL")
	}
	newScript2, err := newScript.Clone()
	if err != nil {
		return errs.Wrap(err, "Could not clone buildscript")
	}
	newScript2.SetProject(url)

	sb, err := newScript2.Marshal()
	if err != nil {
		return errs.Wrap(err, "Could not marshal build script")
	}

	logging.Debug("Writing build script")
	if err := fileutils.WriteFile(filepath.Join(proj.Dir(), constants.BuildScriptFileName), sb); err != nil {
		return errs.Wrap(err, "Could not write build script to file")
	}
	return nil
}
