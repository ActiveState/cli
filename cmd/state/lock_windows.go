// +build windows

package main

import "fmt"

func legacyInstallCommand(branch, version string, isLegacy bool) string {
	prefix := ""
	if isLegacy {
		prefix = "legacy-"
	}
	return fmt.Sprintf(`powershell -Command "& $([scriptblock]::Create((New-Object Net.WebClient).DownloadString('https://platform.activestate.com/dl/cli/%sinstall.ps1'))) -v %s -b %s"`, prefix, version, branch)
}
