// +build linux darwin

package main

import "fmt"

func versionInstallCommand(branch, version string, isLegacy bool) string {
	prefix := ""
	if isLegacy {
		prefix = "legacy-"
	}
	return fmt.Sprintf(`sh <(curl -q https://platform.activestate.com/dl/cli/%sinstall.sh) -v %s -b %s`, prefix, version, branch)
}
