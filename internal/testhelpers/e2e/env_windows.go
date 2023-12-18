//go:build windows
// +build windows

package e2e

import (
	"fmt"
	"os"
)

const (
	testPath             = `C:\Windows\System32;C:\Windows;C:\Windows\System32\Wbem;C:\Windows\System32\WindowsPowerShell\v1.0\;C:\Program Files\PowerShell\7\;`
	systemHomeEnvVarName = "USERPROFILE"
)

func platformEnv(dirs *Dirs) []string {
	return []string{
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
		`PSModuleAnalysisCachePath=C:\PSModuleAnalysisCachePath\ModuleAnalysisCache`,
		`WINDIR=C:\Windows`,
		`PUBLIC=C:\Users\Public`,
		fmt.Sprintf("HOMEPATH=%s", dirs.HomeDir),
		// Other environment variables are commonly set by CI systems, but this one is not.
		// This is requried for some tests in order to get the correct powershell output.
		fmt.Sprintf("PSModulePath=%s", os.Getenv("PSModulePath")),
	}
}
