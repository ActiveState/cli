package orderfile

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	"gopkg.in/yaml.v2"
)

type ErrOrderFileNotExist struct{ *locale.LocalizedError }

func IsErrOrderFileDoesNotExist(err error) bool {
	return errs.Matches(err, &ErrOrderFileNotExist{})
}

type File struct {
	path   string
	script *model.BuildScript
}

func New(path string, script *model.BuildScript) *File {
	return &File{
		path:   path,
		script: script,
	}
}

func Create(path string, script *model.BuildScript) (*File, error) {
	if script == nil {
		script = model.NewBuildScript()
	}

	data, err := yaml.Marshal(script)
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal build script")
	}

	if err := fileutils.WriteFile(path, data); err != nil {
		return nil, errs.Wrap(err, "Could not write build script to file")
	}

	return New(path, script), nil
}

func (o *File) Write() error {
	data, err := yaml.Marshal(o.script)
	if err != nil {
		return errs.Wrap(err, "Could not marshal build script")
	}

	if err := fileutils.WriteFile(o.path, data); err != nil {
		return errs.Wrap(err, "Could not write build script to file")
	}

	return nil
}

func FromPath(path string) (*File, error) {
	data, err := fileutils.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &ErrOrderFileNotExist{locale.NewError("err_orderfile_not_exist", "Order file does not exist at {{.V0}}", path)}
		}
		return nil, errs.Wrap(err, "Could not read build script")
	}

	var script *model.BuildScript
	if err := yaml.Unmarshal(data, &script); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build script")
	}

	return New(path, script), nil
}

func GetOrderFilePathFromWorkingDir() (string, error) {
	root, err := osutils.Getwd()
	if err != nil {
		return "", errs.Wrap(err, "osutils.Getwd failed")
	}

	path, err := fileutils.FindFileInPath(root, constants.OrderFileName)
	if err != nil && !errors.Is(err, fileutils.ErrorFileNotFound) {
		return "", errs.Wrap(err, "fileutils.FindFileInPath %s failed", root)
	}

	return path, nil
}

func (o *File) Update(script *model.BuildScript) error {
	// TODO: Should this only update if it needs to? Or leave it to the caller
	o.script = script
	return o.Write()
}

func (o *File) Path() string {
	return o.path
}

func (o *File) Script() *model.BuildScript {
	return o.script
}

func (o *File) Equals(other *File) bool {
	return o.script.Equals(other.Script())
}
