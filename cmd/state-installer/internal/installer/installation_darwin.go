// +build darwin

package installer

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

const (
	appName = "ActiveState Desktop.app"
)

// InstallSystemFiles installs files in the /Application directory
func InstallSystemFiles(fromDir, binaryDir, systemInstallPath string) error {
	err := os.RemoveAll(filepath.Join(systemInstallPath, appName))
	if err != nil {
		return errs.Wrap(err, "Could not remove old app directory")
	}

	err = fileutils.MoveAllFilesCrossDisk(fromDir, systemInstallPath)
	if err != nil {
		return errs.Wrap(err, "Could not create application directory")
	}

	fromTray := appinfo.TrayApp(binaryDir)
	toTray := appinfo.TrayApp(filepath.Join(systemInstallPath, appName, "Contents", "MacOS"))
	err = createNewSymlink(fromTray.Exec(), toTray.Exec())
	if err != nil {
		return errs.Wrap(err, "Could not create state-tray symlink")
	}

	return nil
}

func RemoveSystemFiles(systemInstallPath string) error {
	return os.RemoveAll(filepath.Join(systemInstallPath, appName))
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
