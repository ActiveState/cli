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
	"github.com/ActiveState/cli/pkg/project"
)

type targeter interface { // note: cannot import runtime/setup.Targeter due to import cycle
	ProjectDir() string
	Owner() string
	Name() string
}

func NewScriptFromProject(proj *project.Project, auth *authentication.Auth) (*Script, error) {
	return newScriptFromFile(filepath.Join(proj.Dir(), constants.BuildScriptFileName), proj.Owner(), proj.Name(), auth)
}

func NewScriptFromTarget(target targeter, auth *authentication.Auth) (*Script, error) {
	return newScriptFromFile(filepath.Join(target.ProjectDir(), constants.BuildScriptFileName), target.Owner(), target.Name(), auth)
}

func newScriptFromFile(path, org, project string, auth *authentication.Auth) (*Script, error) {
	if data, err := fileutils.ReadFile(path); err == nil {
		return NewScript(data)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, errs.Wrap(err, "Could not read build script")
	}

	logging.Debug("Build script does not exist. Creating one.")
	commitId, err := localcommit.Get(filepath.Dir(path))
	if err != nil {
		return nil, errs.Wrap(err, "Unable to get the local commit ID")
	}
	buildplanner := model.NewBuildPlannerModel(auth)
	expr, err := buildplanner.GetBuildExpression(org, project, commitId.String())
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

func Update(proj *project.Project, newExpr *buildexpression.BuildExpression, auth *authentication.Auth) error {
	if script, err := NewScriptFromProject(proj, auth); err == nil && !script.EqualsBuildExpression(newExpr) {
		update(proj.Dir(), newExpr, auth)
	} else if err != nil {
		return errs.Wrap(err, "Could not read build script")
	}
	return nil
}

func UpdateFromTarget(target targeter, newExpr *buildexpression.BuildExpression, auth *authentication.Auth) error {
	return update(target.ProjectDir(), newExpr, auth)
}

func update(projectDir string, newExpr *buildexpression.BuildExpression, auth *authentication.Auth) error {
	script, err := NewScriptFromBuildExpression(newExpr)
	if err != nil {
		return errs.Wrap(err, "Could not parse build expression")
	}

	logging.Debug("Writing build script")
	if err := fileutils.WriteFile(filepath.Join(projectDir, constants.BuildScriptFileName), []byte(script.String())); err != nil {
		return errs.Wrap(err, "Could not write build script to file")
	}
	return nil
}
