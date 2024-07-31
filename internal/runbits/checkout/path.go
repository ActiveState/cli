package checkout

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/project"
)

func (r *Checkout) pathToUse(namespace *project.Namespaced, preferredPath string) (string, error) {
	if preferredPath == "" && namespace == nil {
		return "", errs.New("No namespace or path provided")
	}

	path := preferredPath
	if path == "" {
		logging.Debug("No path provided, using default")

		// Get path from working directory
		wd, err := osutils.Getwd()
		if err != nil {
			return "", errs.Wrap(err, "Could not get working directory")
		}
		path = filepath.Join(wd, namespace.Project)
	}

	if err := validatePath(namespace, path); err != nil {
		return "", errs.Wrap(err, "Validation failed")
	}

	return path, nil
}

type ErrAlreadyCheckedOut struct {
	Path string
}

func (e ErrAlreadyCheckedOut) Error() string {
	return "already checked out"
}

type ErrNoPermission struct {
	Path string
}

func (e ErrNoPermission) Error() string {
	return "no permission"
}

func validatePath(ns *project.Namespaced, path string) error {
	if !fileutils.TargetExists(path) {
		return nil
	}

	empty, err := fileutils.IsEmptyDir(path)
	if err != nil {
		return locale.WrapError(err, "err_namespace_empty_dir", "Could not verify if directory '{{.V0}}' is empty", path)
	}
	if empty {
		return nil
	}

	configFile := filepath.Join(path, constants.ConfigFileName)
	if fileutils.FileExists(configFile) {
		return &ErrAlreadyCheckedOut{path}
	}

	return nil
}
