package buildscript

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
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

func UpdateOrCreate(dir string, newScript *model.BuildExpression) error {
	// If a build script exists, check to see if an update is needed.
	script, err := NewScriptFromProjectDir(dir)
	if err != nil && !IsDoesNotExistError(err) {
		return errs.Wrap(err, "Could not read build script")
	}
	if script != nil && script.Equals(newScript) {
		return nil
	}

	logging.Debug("Writing build script")
	//TODO: enable in DX-1858
	//err := fileutils.WriteFile(path, []byte(newScript.String()))
	//if err != nil {
	//return errs.Wrap(err, "Could not write build script to file")
	//}
	return nil
}
