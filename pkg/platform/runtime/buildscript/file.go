package buildscript

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/go-openapi/strfmt"

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

var ErrBuildscriptNotExist = errors.New("Build script does not exist")

// ScriptFromProjectWithFallback will source the buildscript from the project, and create it if it does not exist.
func ScriptFromProjectWithFallback(proj projecter, auth *authentication.Auth) (*Script, error) {
	path := filepath.Join(proj.ProjectDir(), constants.BuildScriptFileName)

	script, err := ScriptFromFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, errs.Wrap(err, "Could not read build script from file")
		}

		logging.Debug("Build script does not exist. Creating one.")
		commitId, err := localcommit.Get(filepath.Dir(path))
		if err != nil {
			return nil, errs.Wrap(err, "Unable to get the local commit ID")
		}
		buildplanner := model.NewBuildPlannerModel(auth)
		expr, atTime, err := buildplanner.GetBuildExpressionAndTime(commitId.String())
		if err != nil {
			return nil, errs.Wrap(err, "Unable to get the remote build expression and time")
		}
		script, err = NewFromCommit(atTime, expr)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to convert build expression to build script")
		}
		err = fileutils.WriteFile(path, []byte(script.String()))
		if err != nil {
			return nil, errs.Wrap(err, "Unable to write build script")
		}
	}

	return script, nil
}

func ScriptFromProject(proj projecter) (*Script, error) {
	path := filepath.Join(proj.ProjectDir(), constants.BuildScriptFileName)
	return ScriptFromFile(path)
}

func ScriptFromFile(path string) (*Script, error) {
	data, err := fileutils.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errs.Pack(err, ErrBuildscriptNotExist)
		}
		return nil, errs.Wrap(err, "Could not read build script from file")
	}
	return New(data)
}

func Update(proj projecter, atTime *strfmt.DateTime, newExpr *buildexpression.BuildExpression, auth *authentication.Auth) error {
	script, err := ScriptFromProjectWithFallback(proj, auth)
	if err != nil {
		return errs.Wrap(err, "Could not read build script")
	}

	newScript, err := NewFromCommit(atTime, newExpr)
	if err != nil {
		return errs.Wrap(err, "Could not construct new build script to write")
	}

	if script != nil && script.Equals(newScript) {
		return nil // no changes to write
	}

	logging.Debug("Writing build script")
	if err := fileutils.WriteFile(filepath.Join(proj.ProjectDir(), constants.BuildScriptFileName), []byte(newScript.String())); err != nil {
		return errs.Wrap(err, "Could not write build script to file")
	}
	return nil
}
