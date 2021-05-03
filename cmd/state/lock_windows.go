// +build windows

package main

import "fmt"

func legacyInstallCommand(branch, version string) string {
	return fmt.Sprintf(`powershell -Command "& $([scriptblock]::Create((New-Object Net.WebClient).DownloadString('https://platform.activestate.com/dl/cli/install.ps1'))) -v %s -b %s"`, version, branch)
}
