package checkout

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
)

func ensureProjectPath(cfg *config.Instance, namespace *project.Namespaced, preferredPath string) (string, error) {
	targetPath := preferredPath
	if targetPath == "" {
		var err error
		targetPath, err = getProjectPath(cfg, namespace)
		if err != nil {
			return "", errs.Wrap(err, "Could not get project path")
		}
	}

	err := fileutils.MkdirUnlessExists(targetPath)
	if err != nil {
		return "", errs.Wrap(err, "Could not make directory at: %s", targetPath)
	}

	// Validate that target path doesn't contain a config for a different namespace
	if err := validatePath(namespace.Project, targetPath); err != nil {
		return "", errs.Wrap(err, "Could not validate target path: %s", targetPath)
	}

	return targetPath, nil
}

func getProjectPath(config *config.Instance, namespace *project.Namespaced) (string, error) {
	paths := projectfile.GetProjectPaths(config, namespace.String())
	if len(paths) > 0 {
		return paths[0], nil
	}

	targetPath, err := getSafeWorkDir()
	if err != nil {
		return "", locale.NewError("err_get_wd")
	}

	return filepath.Join(targetPath, namespace.Project), nil
}

func validatePath(name string, path string) error {
	empty, err := fileutils.IsEmptyDir(path)
	if err != nil {
		return locale.WrapError(err, "err_namespace_empty_dir", "Could not verify if directory is empty")
	}
	if empty {
		return nil
	}

	configFile := filepath.Join(path, constants.ConfigFileName)
	if !fileutils.FileExists(configFile) {
		// Directory is not empty and does not contain a config file
		return locale.NewError("err_directory_in_use")
	}

	pj, err := project.Parse(configFile)
	if err != nil {
		return locale.WrapError(err, "err_parse_project", "", configFile)
	}

	if !pj.IsHeadless() && pj.Name() != name {
		return locale.NewInputError("err_target_path_namespace_match", "", name, pj.Name())
	}

	return nil
}

func getSafeWorkDir() (string, error) {
	dir, err := osutils.Getwd()
	if err != nil {
		return "", errs.Wrap(err, "Could not get working directory")
	}

	if !strings.HasPrefix(strings.ToLower(dir), `c:\windows`) {
		return dir, nil
	}

	dir, err = os.UserHomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}

	return dir, nil
}
