package legacytray

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
)

const macOSApplicationName = "ActiveState Desktop (Preview).app"
const trayLaunchFileName = "state-tray"

func osSpecificRemoveTray(installPath, trayExec string) error {
	if launcherPath, err := installation.ApplicationInstallPath(); err == nil {
		if appPath := filepath.Join(launcherPath, macOSApplicationName); fileutils.DirExists(appPath) {
			err = os.RemoveAll(appPath)
			if err != nil {
				return errs.Wrap(err, "Unable to remove launcher app")
			}
		}
	}

	// The system directory is on MacOS only and contains the tray
	// application files. It is safe for us to remove this directory
	// without first inspecting the contents.
	if systemDir := filepath.Join(installPath, "system"); fileutils.DirExists(systemDir) {
		err := os.RemoveAll(systemDir)
		if err != nil {
			return errs.Wrap(err, "Unable to remove system dir")
		}
	}

	return nil
}
