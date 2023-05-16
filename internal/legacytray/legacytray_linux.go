package legacytray

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils/user"
)

const trayIconFileName = "state-tray.svg"
const trayLaunchFileName = "state-tray.desktop"

func osSpecificRemoveTray(installPath, trayExec string) error {
	// Remove any .desktop files and icons installed.
	if homeDir, err := user.HomeDir(); err == nil {
		appDir := filepath.Join(homeDir, constants.ApplicationDir)
		if desktopFile := filepath.Join(appDir, trayLaunchFileName); fileutils.FileExists(desktopFile) {
			err = os.Remove(desktopFile)
			if err != nil {
				return errs.Wrap(err, "Unable to remove desktop file")
			}
		}
		iconsDir := filepath.Join(homeDir, constants.IconsDir)
		if iconFile := filepath.Join(iconsDir, trayIconFileName); fileutils.FileExists(iconFile) {
			err = os.Remove(iconFile)
			if err != nil {
				return errs.Wrap(err, "Unable to remove icon file")
			}
		}
	}
	return nil
}
