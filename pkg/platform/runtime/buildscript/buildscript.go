package buildscript

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	"gopkg.in/yaml.v2"
)

type DoesNotExistError struct{ *locale.LocalizedError }

func IsDoesNotExistError(err error) bool {
	return errs.Matches(err, &DoesNotExistError{})
}

type File struct {
	Path   string
	Script *model.BuildScript
}

func New(path string, script *model.BuildScript) *File {
	return &File{path, script}
}

func Create(path string, script *model.BuildScript) (*File, error) {
	if script == nil {
		script = model.NewBuildScript()
	}

	data, err := yaml.Marshal(script)
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal build script")
	}

	buildScriptPath := filepath.Join(path, constants.BuildScriptFileName)
	if err := fileutils.WriteFile(buildScriptPath, data); err != nil {
		return nil, errs.Wrap(err, "Could not write build script to file")
	}

	return New(buildScriptPath, script), nil
}

func (o *File) Write() error {
	data, err := yaml.Marshal(o.Script)
	if err != nil {
		return errs.Wrap(err, "Could not marshal build script")
	}

	if err := fileutils.WriteFile(o.Path, data); err != nil {
		return errs.Wrap(err, "Could not write build script to file")
	}

	return nil
}

func FromPath(path string) (*File, error) {
	buildScriptPath := filepath.Join(path, constants.BuildScriptFileName)
	data, err := fileutils.ReadFile(buildScriptPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &DoesNotExistError{locale.NewError("err_build_script_not_exist", "Build script does not exist at {{.V0}}", path)}
		}
		return nil, errs.Wrap(err, "Could not read build script")
	}

	var script *model.BuildScript
	if err := yaml.Unmarshal(data, &script); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build script")
	}

	return New(buildScriptPath, script), nil
}

func (o *File) Update(script *model.BuildScript) error {
	if o.Script.Equals(script) {
		return nil
	}

	o.Script = script
	return o.Write()
}
