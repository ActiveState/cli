package checkout

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
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
		wd, err := os.Getwd()
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
	if !fileutils.FileExists(configFile) {
		// Directory is not empty and does not contain a config file
		return locale.NewInputError("err_directory_in_use", "", path)
	}

	pj, err := project.Parse(configFile)
	if err != nil {
		return locale.WrapError(err, "err_parse_project", "", configFile)
	}

	pjns := pj.Namespace()
	if ns != nil && ns.IsValid() && !pj.IsHeadless() && (pjns.Owner != ns.Owner || pjns.Project != ns.Project) {
		return locale.NewInputError("err_target_path_namespace_match", "", ns.String(), pjns.String())
	}

	return nil
}
