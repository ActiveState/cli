package buildscript

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
)

type DoesNotExistError struct{ error }

func IsDoesNotExistError(err error) bool {
	return errs.Matches(err, &DoesNotExistError{})
}

func NewScriptFromProjectDir(dir string) (*Script, error) {
	return newScriptFromFile(filepath.Join(dir, constants.BuildScriptFileName))
}

func newScriptFromFile(path string) (*Script, error) {
	data, err := fileutils.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &DoesNotExistError{errs.New("Build script '%s' does not exist", path)}
		}
		return nil, errs.Wrap(err, "Could not read build script")
	}
	return NewScript(data)
}

func UpdateOrCreate(dir string, newExpr *buildexpression.BuildExpression) error {
	// If a build script exists, check to see if an update is needed.
	script, err := NewScriptFromProjectDir(dir)
	if err != nil && !IsDoesNotExistError(err) {
		return errs.Wrap(err, "Could not read build script")
	}
	if script != nil && script.EqualsBuildExpression(newExpr) {
		return nil
	}

	data, err := json.Marshal(newExpr)
	if err != nil {
		return errs.Wrap(err, "Could not marshal buildexpression to JSON")
	}
	script, err = NewScriptFromBuildExpression(data)
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
