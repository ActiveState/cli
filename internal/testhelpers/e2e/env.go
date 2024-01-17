package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/require"
)

func sandboxedTestEnvironment(t *testing.T, dirs *Dirs, updatePath bool, extraEnv ...string) []string {
	var env []string
	basePath := platformPath()
	if os.Getenv(constants.OverrideSandbox) != "" {
		basePath = os.Getenv("PATH")
		env = append(env, os.Environ()...)
	}
	if value := os.Getenv(constants.ActiveStateCIEnvVarName); value != "" {
		env = append(env, fmt.Sprintf("%s=%s", constants.ActiveStateCIEnvVarName, value))
	}

	// add go binary to PATH
	goBinary := goBinaryPath(t)
	basePath = fmt.Sprintf("%s%s%s", basePath, string(os.PathListSeparator), filepath.Dir(goBinary))

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

	if updatePath {
		// add bin path
		oldPath := basePath
		newPath := fmt.Sprintf(
			"PATH=%s%s%s",
			dirs.Bin, string(os.PathListSeparator), oldPath,
		)
		env = append(env, newPath)
	} else {
		env = append(env, "PATH="+basePath)
	}

	// append platform specific environment variables
	env = append(env, platformSpecificEnv(dirs)...)

	// Prepare sandboxed home directory
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
	err := fileutils.Touch(rcFile)
	if err != nil {
		return errs.Wrap(err, "Could not create rc file")
	}

	return nil
}

func goBinaryPath(t *testing.T) string {
	locator := "which"
	if runtime.GOOS == "windows" {
		locator = "where"
	}
	cmd := exec.Command(locator, "go")
	output, err := cmd.Output()
	if err != nil {
		t.Log("Could not find go binary")
		return ""
	}
	goBinary := string(output)
	goBinary = strings.TrimSpace(string(goBinary))
	return goBinary
}
