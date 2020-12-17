// +build !windows

package clean

import (
	"os"
)

func removeInstall(installPath string) error {
	return os.Remove(installPath)
}
