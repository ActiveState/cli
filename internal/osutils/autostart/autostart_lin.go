//+build linux

package autostart

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/gobuffalo/packr"
	"github.com/mitchellh/go-homedir"
)

func (a *App) Enable() error {
	appPath, err := ensuredAppPath()
	if err != nil {
		return errs.Wrap(err, "Could not ensure application file is present")
	}

	enabled, err := a.IsEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}
	if enabled {
		return nil
	}

	dir, err := dirPath(autostartDir)
	if err != nil {
		return errs.Wrap(err, "Could not get autostart file")
	}
	path := filepath.Join(dir, launchFile)

	if err = os.Symlink(appPath, path); err != nil {
		return errs.Wrap(err, "Could not create symlink")
	}
	return nil
}

func (a *App) Disable() error {
	enabled, err := a.IsEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}
	if !enabled {
		return nil
	}

	dir, err := dirPath(autostartDir)
	if err != nil {
		return errs.Wrap(err, "Could not get autostart file")
	}
	path := filepath.Join(dir, launchFile)

	return os.Remove(path)
}

func (a *App) IsEnabled() (bool, error) {
	dir, err := dirPath(autostartDir)
	if err != nil {
		return false, errs.Wrap(err, "Could not get autostart file")
	}
	path := filepath.Join(dir, launchFile)

	return fileutils.FileExists(path), nil
}

const (
	applicationDir = ".local/share/applications"
	autostartDir   = ".config/autostart"
	iconsDir       = ".local/share/icons/hicolor/scalable/apps"
	launchFile     = "state-tray.desktop"
	iconFileState  = "state-tray.svg"
	iconFileBase   = "icon.svg"
)

func dirPath(dir string) (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(homeDir, dir), nil
}

func ensuredAppPath() (string, error) {
	appDir, err := dirPath(applicationDir)
	if err != nil {
		return "", errs.Wrap(err, "Could not get application file")
	}
	appPath := filepath.Join(appDir, launchFile)

	if !fileutils.FileExists(appPath) {
		box := packr.NewBox("../../../assets")

		icons, err := dirPath(iconsDir)
		if err != nil {
			return "", errs.Wrap(err, "Could not get icons directory")
		}
		iconPath := filepath.Join(icons, iconFileState)

		if err := fileutils.WriteFile(iconPath, box.Bytes(iconFileBase)); err != nil {
			return "", errs.Wrap(err, "Could not write icon file")
		}

		if err := fileutils.WriteFile(appPath, box.Bytes(launchFile)); err != nil {
			return "", errs.Wrap(err, "Could not write application file")
		}

		file, err := os.Open(appPath)
		if err != nil {
			return "", errs.Wrap(err, "Could not open application file")
		}
		err = file.Chmod(0770)
		file.Close()
		if err != nil {
			return "", errs.Wrap(err, "Could not make file executable")
		}

		cmd := exec.Command("gio", "set", appPath, "metadata::trusted", "true")
		if err := cmd.Run(); err != nil {
			return "", errs.Wrap(err, "Could not set application file as trusted")
		}
	}

	return appPath, nil
}
