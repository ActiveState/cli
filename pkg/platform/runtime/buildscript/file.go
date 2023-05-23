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

	"github.com/alecthomas/participle/v2"
	"gopkg.in/yaml.v2"
)

type DoesNotExistError struct{ error }

func IsDoesNotExistError(err error) bool {
	return errs.Matches(err, &DoesNotExistError{})
}

type File struct {
	Path   string
	Script *Script
}

func get(path string) (*File, error) {
	data, err := fileutils.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &DoesNotExistError{errs.New("Build script '%s' does not exist", path)}
		}
		return nil, errs.Wrap(err, "Could not read build script")
	}

	parser, err := participle.Build[Script]()
	if err != nil {
		return nil, errs.Wrap(err, "Could not create parser for build script")
	}

	script, err := parser.ParseBytes(constants.BuildScriptFileName, data)
	if err != nil {
		logging.Error("%v", err)
		return nil, errs.Wrap(err, "Could not parse build script")
	}

	return &File{path, script}, nil
}

func Get(dir string) (*File, error) {
	return get(filepath.Join(dir, constants.BuildScriptFileName))
}

func UpdateOrCreate(dir string, script *model.BuildScript) error {
	file, err := Get(dir)
	if err != nil {
		if !IsDoesNotExistError(err) {
			return errs.Wrap(err, "Could not get build script")
		}
		file, err = create(dir, nil)
		if err != nil {
			return errs.Wrap(err, "Could not create build script")
		}
	}
	return file.update(script)
}

func create(dir string, script *model.BuildScript) (*File, error) {
	return nil, errs.New("Writing not supported yet")

	if script == nil {
		script = model.NewBuildScript()
	}

	data, err := yaml.Marshal(script)
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal build script")
	}

	path := filepath.Join(dir, constants.BuildScriptFileName)
	logging.Debug("Creating build script: %s", path)
	if err := fileutils.WriteFile(path, data); err != nil {
		return nil, errs.Wrap(err, "Could not write build script to file")
	}

	//return &File{path, script}, nil
	return nil, errs.New("Writing not supported yet")
}

func (o *File) write() error {
	logging.Debug("Writing build script")
	if err := fileutils.WriteFile(o.Path, []byte(o.Script.String())); err != nil {
		return errs.Wrap(err, "Could not write build script to file")
	}
	return nil
}

func (o *File) update(script *model.BuildScript) error {
	if script == nil {
		return errs.New("Build script to write is nil")
	}
	if o.Script != nil && o.Script.Equals(script) {
		return nil
	}

	//o.Script = script
	return o.write()
}
