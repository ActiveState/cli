//go:build windows
// +build windows

package e2e

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
)

const (
	basePath             = `C:\Windows\System32;C:\Windows;C:\Windows\System32\Wbem;C:\Windows\System32\WindowsPowerShell\v1.0\;C:\Program Files\PowerShell\7\;`
	systemHomeEnvVarName = "USERPROFILE"
)

func platformSpecificEnv(dirs *Dirs) []string {
	env := []string{
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
		"WINDIR=C:\\Windows",
		"PUBLIC=C:\\Users\\Public",
		"PSModuleAnalysisCachePath=C:\\PSModuleAnalysisCachePath\\ModuleAnalysisCache",
		fmt.Sprintf("HOMEPATH=%s", dirs.HomeDir),
		// Other environment variables are commonly set by CI systems, but this one is not.
		// This is requried for some tests in order to get the correct powershell output.
		fmt.Sprintf("PSModulePath=%s", os.Getenv("PSModulePath")),
		fmt.Sprintf("LOCALAPPDATA=%s", dirs.TempDir),
		fmt.Sprintf("%s=true", constants.DisableActivateEventsEnvVarName),
	}

	if condition.OnCI() {
		env = append(env, fmt.Sprintf("%s=cmd.exe", constants.OverrideShellEnvVarName))
	}

	return env
}

func platformPath() string {
	if condition.OnCI() {
		return `C:\msys64\usr\bin` + string(os.PathListSeparator) + basePath
	}
	return basePath
}
