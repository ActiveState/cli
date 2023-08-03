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
)

// projecter is a union between project.Project and setup.Targeter
type projecter interface {
	ProjectDir() string
	Owner() string
	Name() string
}

func NewScriptFromProject(proj projecter, auth *authentication.Auth) (*Script, error) {
	return newScriptFromFile(filepath.Join(proj.ProjectDir(), constants.BuildScriptFileName), proj.Owner(), proj.Name(), auth)
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

func Update(proj projecter, newExpr *buildexpression.BuildExpression, auth *authentication.Auth) error {
	if script, err := NewScriptFromProject(proj, auth); err == nil && (script == nil || !script.EqualsBuildExpression(newExpr)) {
		update(proj.ProjectDir(), newExpr, auth)
	} else if err != nil {
		return errs.Wrap(err, "Could not read build script")
	}
	return nil
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
