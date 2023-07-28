package buildscript

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func NewScriptFromProjectDir(dir string, auth *authentication.Auth) (*Script, error) {
	return newScriptFromFile(filepath.Join(dir, constants.BuildScriptFileName), auth)
}

func newScriptFromFile(path string, auth *authentication.Auth) (*Script, error) {
	if data, err := fileutils.ReadFile(path); err == nil {
		return NewScript(data)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, errs.Wrap(err, "Could not read build script")
	}

	logging.Debug("Build script does not exist. Creating one.")
	dir := filepath.Dir(path)
	pjf, err := projectfile.FromPath(dir)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to read the project")
	}
	commitId, err := localcommit.Get(dir)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to get the local commit ID")
	}
	buildplanner := model.NewBuildPlannerModel(auth)
	expr, err := buildplanner.GetBuildExpression(pjf.Owner(), pjf.Name(), commitId.String())
	if err != nil {
		return nil, errs.Wrap(err, "Unable to get the remote build expression")
	}
	script, err := NewScriptFromBuildExpression(expr)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to convert build expression to build script")
	}
	err = fileutils.WriteFile(path, []byte(script.String()))
	if err != nil {
		return nil, errs.Wrap(err, "Unable to write build script")
	}
	return script, nil
}

func UpdateOrCreate(dir string, newExpr *buildexpression.BuildExpression, auth *authentication.Auth) error {
	// If a build script exists, check to see if an update is needed.
	script, err := NewScriptFromProjectDir(dir, auth)
	if err != nil {
		return errs.Wrap(err, "Could not read build script")
	}
	if script.EqualsBuildExpression(newExpr) {
		return nil
	}

	script, err = NewScriptFromBuildExpression(newExpr)
	if err != nil {
		return errs.Wrap(err, "Could not parse build expression")
	}

	logging.Debug("Writing build script")
	err = fileutils.WriteFile(filepath.Join(dir, constants.BuildScriptFileName), []byte(script.String()))
	if err != nil {
		return errs.Wrap(err, "Could not write build script to file")
	}
	return nil
}
