package buildscript

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	"gopkg.in/yaml.v2"
)

type DoesNotExistError struct{ error }

func IsDoesNotExistError(err error) bool {
	return errs.Matches(err, &DoesNotExistError{})
}

type File struct {
	Path   string
	Script *model.BuildScript
}

func Get(dir string) (*File, error) {
	path := filepath.Join(dir, constants.BuildScriptFileName)
	data, err := fileutils.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &DoesNotExistError{errs.New("Build script does not exist at %s", dir)}
		}
		return nil, errs.Wrap(err, "Could not read build script")
	}

	var script *model.BuildScript
	if err := yaml.Unmarshal(data, &script); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build script")
	}

	return &File{path, script}, nil
}

func UpdateOrCreate(dir string, script *model.BuildScript) error {
	of, err := Get(dir)
	if err != nil {
		if !IsDoesNotExistError(err) {
			return errs.Wrap(err, "Could not get build script")
		}
		_, err2 := create(dir, nil)
		if err2 != nil {
			return errs.Wrap(err2, "Could not create build script")
		}
	}
	return of.Update(script)
}

func create(dir string, script *model.BuildScript) (*File, error) {
	if script == nil {
		script = model.NewBuildScript()
	}

	data, err := yaml.Marshal(script)
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal build script")
	}

	path := filepath.Join(dir, constants.BuildScriptFileName)
	if err := fileutils.WriteFile(path, data); err != nil {
		return nil, errs.Wrap(err, "Could not write build script to file")
	}

	return &File{path, script}, nil
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

func (o *File) Update(script *model.BuildScript) error {
	if o.Script.Equals(script) {
		return nil
	}

	o.Script = script
	return o.Write()
}
