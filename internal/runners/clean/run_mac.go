//+build darwin

package clean

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/mitchellh/go-homedir"
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

func autostartFilePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", locale.WrapError(err, "err_clean_get_home", "Could not get home directory")
	}

	return filepath.Join(home, "Library/LaunchAgents", "com.activestate.platform.state-tray.plist"), nil
}
