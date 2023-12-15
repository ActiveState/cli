package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/require"
)

func sandboxedTestEnvironment(t *testing.T, dirs *Dirs, updatePath bool, extraEnv ...string) []string {
	var env []string
	env = append(env, []string{
		constants.ConfigEnvVarName + "=" + dirs.Config,
		constants.CacheEnvVarName + "=" + dirs.Cache,
		constants.DisableRuntime + "=true",
		constants.ProjectEnvVarName + "=",
		constants.E2ETestEnvVarName + "=true",
		constants.DisableUpdates + "=true",
		constants.DisableProjectMigrationPrompt + "=true",
		constants.OptinUnstableEnvVarName + "=true",
		constants.ServiceSockDir + "=" + dirs.SockRoot,
		constants.HomeEnvVarName + "=" + dirs.HomeDir,
		systemHomeEnvVarName + "=" + dirs.HomeDir,
		"NO_COLOR=true",
		"CI=true",
	}...)

	path := testPath
	if runtime.GOOS == "windows" {
		path = os.Getenv("PATH")
		env = append(env, os.Environ()...)
	}

	if updatePath {
		// add bin path
		// Remove release state tool installation from PATH
		oldPath := path
		newPath := fmt.Sprintf(
			"PATH=%s%s%s",
			dirs.Bin, string(os.PathListSeparator), oldPath,
		)
		env = append(env, newPath)
	} else {
		env = append(env, "PATH="+path)
	}

	err := prepareHomeDir(dirs.HomeDir)
	require.NoError(t, err)

	// add session environment variables
	env = append(env, extraEnv...)

	return env
}

func prepareHomeDir(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	if !fileutils.DirExists(dir) {
		err := fileutils.Mkdir(dir)
		if err != nil {
			return errs.Wrap(err, "Could not create home dir")
		}
	}

	var filename string
	switch runtime.GOOS {
	case "linux":
		filename = ".bashrc"
	case "darwin":
		filename = ".zshrc"
	}

	rcFile := filepath.Join(dir, filename)
	fmt.Println("Creating rc file: " + rcFile)
	err := fileutils.Touch(rcFile)
	if err != nil {
		return errs.Wrap(err, "Could not create rc file")
	}

	return nil

}
