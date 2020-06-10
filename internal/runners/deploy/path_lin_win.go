// +build !darwin

package deploy

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/thoas/go-funk"
)

// usablePath will find a writable directory under PATH
func usablePath() (string, error) {
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	if len(paths) == 0 {
		return "", locale.NewInputError("err_deploy_path_empty", "Your system does not have any PATH entries configured, so symlinks can not be created.")
	}

	preferredPaths := []string{
		"/usr/local/bin",
		"/usr/bin",
	}
	var result string
	for _, path := range paths {
		if path == "" || (!fileutils.IsDir(path) && !fileutils.FileExists(path)) || !fileutils.IsWritable(path) {
			continue
		}

		// Record result
		if funk.Contains(preferredPaths, path) {
			return path, nil
		}
		result = path
	}

	if result != "" {
		return result, nil
	}

	return "", locale.NewInputError("err_deploy_path_noperm", "No permission to create symlinks on any of the PATH entries: {{.V0}}.", os.Getenv("PATH"))
}
