package orderfile

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

	orderFilePath := filepath.Join(path, constants.OrderFileName)
	if err := fileutils.WriteFile(orderFilePath, data); err != nil {
		return nil, errs.Wrap(err, "Could not write build script to file")
	}

	return New(orderFilePath, script), nil
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
	orderFilePath := filepath.Join(path, constants.OrderFileName)
	data, err := fileutils.ReadFile(orderFilePath)
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

	return New(orderFilePath, script), nil
}

func (o *File) Update(script *model.BuildScript) error {
	if o.script.Equals(script) {
		return nil
	}

	o.script = script
	return o.Write()
}

func (o *File) Path() string {
	return o.path
}

func (o *File) Script() *model.BuildScript {
	return o.script
}
