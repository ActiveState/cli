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

// configurer is here until buildscripts are no longer walled behind an opt-in config option.
type configurer interface {
	GetBool(string) bool
}

var ErrBuildscriptNotExist = errors.New("Build script does not exist")

func ScriptFromProject(projectDir string) (*buildscript.BuildScript, error) {
	path := filepath.Join(projectDir, constants.BuildScriptFileName)

	data, err := fileutils.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errs.Pack(err, ErrBuildscriptNotExist)
		}
		return nil, errs.Wrap(err, "Could not read build script from file")
	}

	script, err := buildscript.Unmarshal(data)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build script")
	}

	// Synchronize any changes with activestate.yaml.
	err = checkoutinfo.UpdateProject(script, path)
	if err != nil {
		return nil, errs.Wrap(err, "Could not update project file")
	}

	return script, nil
}

func Initialize(path, owner, project, branch, commitID string, auth *authentication.Auth, cfg configurer) error {
	if cfg.GetBool(constants.OptinBuildscriptsConfig) {
		_, err := ScriptFromProject(path)
		if err == nil {
			return nil // nothing to do, buildscript already exists
		}
		if !errors.Is(err, os.ErrNotExist) {
			return errs.Wrap(err, "Could not read project build script")
		}
	}

	buildplanner := buildplanner.NewBuildPlannerModel(auth)
	script, err := buildplanner.GetBuildScript(owner, project, branch, commitID)
	if err != nil {
		return errs.Wrap(err, "Unable to get the remote build script")
	}

	if !cfg.GetBool(constants.OptinBuildscriptsConfig) {
		// Just update the project file with the new commit ID.
		err = checkoutinfo.UpdateProject(script, path)
		if err != nil {
			return errs.Wrap(err, "Unable to update project file")
		}
		return nil
	}

	scriptBytes, err := script.Marshal()
	if err != nil {
		return errs.Wrap(err, "Unable to marshal build script")
	}

	scriptPath := filepath.Join(path, constants.BuildScriptFileName)
	logging.Debug("Initializing build script at %s", scriptPath)
	err = fileutils.WriteFile(scriptPath, scriptBytes)
	if err != nil {
		return errs.Wrap(err, "Unable to write build script")
	}

	return nil
}

func Update(path string, newScript *buildscript.BuildScript, cfg configurer) error {
	if !cfg.GetBool(constants.OptinBuildscriptsConfig) {
		// Just update the activestate.yaml file (e.g. with the new commit ID).
		// Eventually the buildscript will be the one source of truth.
		return checkoutinfo.UpdateProject(newScript, path)
	}

	script, err := ScriptFromProject(path)
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
	if err := fileutils.WriteFile(filepath.Join(path, constants.BuildScriptFileName), sb); err != nil {
		return errs.Wrap(err, "Could not write build script to file")
	}

	// Synchronize changes with activestate.yaml.
	err = checkoutinfo.UpdateProject(newScript, path)
	if err != nil {
		return errs.Wrap(err, "Could not update project file")
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
