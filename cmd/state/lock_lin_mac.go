// +build linux darwin

package main

import "fmt"

func legacyInstallCommand(branch, version string) string {
	return fmt.Sprintf(`sh <(curl -q https://platform.activestate.com/dl/cli/install.sh) -v %s -b %s`, version, branch)
}
