// +build darwin

package installer

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

// InstallSystemFiles installs files in the /Application directory
func InstallSystemFiles(fromDir, binaryDir, systemInstallPath string) error {
	err := os.RemoveAll(filepath.Join(systemInstallPath, constants.MacOSApplicationName))
	if err != nil {
		return errs.Wrap(err, "Could not remove old app directory")
	}

	// ensure systemInstallPath exists
	err = fileutils.MkdirUnlessExists(systemInstallPath)
	if err != nil {
		return errs.Wrap(err, "Application directory %s did not exist, and failed to create it", systemInstallPath)
	}

	err = fileutils.MoveAllFilesCrossDisk(fromDir, systemInstallPath)
	if err != nil {
		return errs.Wrap(err, "Could not create application directory")
	}

	fromTray := appinfo.TrayApp(binaryDir)
	toTray := appinfo.TrayApp(filepath.Join(systemInstallPath, constants.MacOSApplicationName, "Contents", "MacOS"))
	err = createNewSymlink(fromTray.Exec(), toTray.Exec())
	if err != nil {
		return errs.Wrap(err, "Could not create state-tray symlink")
	}

	return nil
}

func createNewSymlink(target, filename string) error {
	if fileutils.TargetExists(filename) {
		if err := os.Remove(filename); err != nil {
			return errs.Wrap(err, "Could not delete existing symlink")
		}
	}

	err := os.Symlink(target, filename)
	if err != nil {
		return errs.Wrap(err, "Could not create symlink")
	}
	return nil
}
