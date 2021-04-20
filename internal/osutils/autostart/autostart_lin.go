//+build linux

package autostart

import (
	"os"
	"os/exec"
	"path"
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

	path, err := filePath(autostartDir)
	if err != nil {
		return errs.Wrap(err, "Could not get autostart file")
	}

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

	path, err := filePath(autostartDir)
	if err != nil {
		return errs.Wrap(err, "Could not get autostart file")
	}
	return os.Remove(path)
}

func (a *App) IsEnabled() (bool, error) {
	path, err := filePath(autostartDir)
	if err != nil {
		return false, errs.Wrap(err, "Could not get autostart file")
	}
	return fileutils.FileExists(path), nil
}

const (
	applicationDir   = ".local/share/applications"
	autostartDir     = ".config/autostart"
	launchFileUbuntu = "state-tray.desktop"
)

func filePath(dir string) (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(homeDir, dir, launchFileUbuntu), nil
}

func ensuredAppPath() (string, error) {
	appPath, err := filePath(applicationDir)
	if err != nil {
		return "", errs.Wrap(err, "Could not get application file")
	}

	if !fileutils.FileExists(appPath) {
		box := packr.NewBox("../../../assets")
		fileName := path.Base(appPath)
		if err := fileutils.WriteFile(appPath, box.Bytes(fileName)); err != nil {
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
