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

func (u *Uninstall) removeInstall() error {
	err := u.removeStateTray()
	if err != nil {
		return err
	}

	return u.removeInstallDir()
}

func (u *Uninstall) removeStateTray() error {
	err := u.removeAutoStartFile()
	if err != nil {
		return err
	}

	return removeTrayApp()
}

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
