package buildscript_runbit

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/checkoutinfo"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
)

// projecter is a union between project.Project and setup.Targeter
type projecter interface {
	ProjectDir() string
	Owner() string
	Name() string
}

var ErrBuildscriptNotExist = errors.New("Build script does not exist")

func ScriptFromProject(proj projecter) (*buildscript.BuildScript, error) {
	path := filepath.Join(proj.ProjectDir(), constants.BuildScriptFileName)
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

func Initialize(path, owner, project, branch string, auth *authentication.Auth) error {
	scriptPath := filepath.Join(path, constants.BuildScriptFileName)
	script, err := ScriptFromFile(scriptPath)
	if err == nil {
		return nil // nothing to do, buildscript already exists
	}
	if !errors.Is(err, os.ErrNotExist) {
		return errs.Wrap(err, "Could not read build script from file")
	}

	logging.Debug("Build script does not exist. Creating one.")
	commitId, err := checkoutinfo.GetCommitID(path)
	if err != nil {
		return errs.Wrap(err, "Unable to get the local commit ID")
	}

	buildplanner := buildplanner.NewBuildPlannerModel(auth)
	script, err = buildplanner.GetBuildScript(owner, project, branch, commitId.String())
	if err != nil {
		return errs.Wrap(err, "Unable to get the remote build expression and time")
	}

	scriptBytes, err := script.Marshal()
	if err != nil {
		return errs.Wrap(err, "Unable to marshal build script")
	}

	logging.Debug("Initializing build script at %s", scriptPath)
	err = fileutils.WriteFile(scriptPath, scriptBytes)
	if err != nil {
		return errs.Wrap(err, "Unable to write build script")
	}

	return nil
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

	sb, err := newScript.Marshal()
	if err != nil {
		return errs.Wrap(err, "Could not marshal build script")
	}

	logging.Debug("Writing build script")
	if err := fileutils.WriteFile(filepath.Join(proj.ProjectDir(), constants.BuildScriptFileName), sb); err != nil {
		return errs.Wrap(err, "Could not write build script to file")
	}
	return nil
}

// Remove removes an existing buildscript if it exists.
// This is primarily for updating an outdated buildscript.
func Remove(path string) error {
	bsPath := filepath.Join(path, constants.BuildScriptFileName)
	if !fileutils.TargetExists(bsPath) {
		return nil
	}
	return os.Remove(bsPath)
}
