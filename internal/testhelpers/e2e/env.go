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
		// path = os.Getenv("PATH")
		// env = append(env, os.Environ()...)
		windowsEnv := []string{
			"SystemDrive=C:",
			"SystemRoot=C:\\Windows",
			"PROGRAMFILES=C:\\Program Files",
			"ProgramFiles(x86)=C:\\Program Files (x86)",
			"PATHEXT=.COM;.EXE;.BAT;.CMD;.VBS;.VBE;.JS;.JSE;.WSF;.WSH;.MSC",
			"HOMEDRIVE=C:",
			"ALLUSERSPROFILE=C:\\ProgramData",
			"ProgramData=C:\\ProgramData",
			"COMSPEC=C:\\Windows\\System32\\cmd.exe",
			"PROGRAMFILES=C:\\Program Files",
			"CommonProgramW6432=C:\\Program Files\\Common Files",
			`PSModulePath=C:\\Modules\azurerm_2.1.0;C:\\Modules\azure_2.1.0;C:\Users\packer\Documents\WindowsPowerShell\Modules;C:\Program Files\WindowsPowerShell\Modules;C:\Windows\system32\WindowsPowerShell\v1.0\Modules;C:\Program Files\Microsoft SQL Server\130\Tools\PowerShell\Modules\;C:\Program Files (x86)\Google\Cloud SDK\google-cloud-sdk\platform\PowerShell`,
			"SHLVL=1",
			fmt.Sprintf("HOME=%s", dirs.HomeDir),
			fmt.Sprintf("HOMEPATH=%s", dirs.HomeDir),
			`PSModuleAnalysisCachePath=C:\PSModuleAnalysisCachePath\ModuleAnalysisCache`,
			`WINDIR=C:\Windows`,
			`PUBLIC=C:\Users\Public`,
		}
		env = append(env, windowsEnv...)
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
