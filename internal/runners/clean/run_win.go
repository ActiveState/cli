// +build windows

package clean

import "github.com/ActiveState/cli/internal/embedrun"

func removeInstall(installPath string) error {
	return embedrun.Script("removeInstall", installPath)
}
