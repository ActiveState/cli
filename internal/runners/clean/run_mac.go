//+build darwin

package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
)

const (
	trayInstallPath = "/Applications/state-tray.app"
)

func removeStateTray(cfg configurable) error {
	autoStartPath := cfg.GetString(constants.AutoStartPath)
	if autoStartPath != "" {
		err := removeAutoStartFile(autoStartPath)
		if err != nil {
			return err
		}
	}
	return removeTray()
}

func removeTray() error {
	err := os.RemoveAll(trayInstallPath)
	if err != nil {
		return locale.WrapError(err, "err_remove_tray", "Could not remove state tray")
	}

	return nil
}
