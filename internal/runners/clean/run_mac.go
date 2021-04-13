//+build darwin

package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
)

const (
	trayAppInstallPath = "/Applications/state-tray.app"
)

func removeTrayApp() error {
	if !fileutils.DirExists(trayAppInstallPath) {
		return nil
	}

	err := os.RemoveAll(trayAppInstallPath)
	if err != nil {
		return locale.WrapError(err, "err_remove_tray", "Could not remove state tray")
	}
	return nil
}
