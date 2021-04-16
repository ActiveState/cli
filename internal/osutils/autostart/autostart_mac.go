// +build darwin

package autostart

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/gobuffalo/packr"
	"github.com/mitchellh/go-homedir"
)

const launchFileMacOS = "com.activestate.platform.state-tray.plist"

func (a *App) Enable() error {
	enabled, err := a.IsEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}

	if enabled {
		return nil
	}

	path, err := launchFilePath()
	if err != nil {
		return errs.Wrap(err, "Could not get launch file")
	}

	box := packr.NewBox("../../../assets")
	err = fileutils.WriteFile(path, box.Bytes(launchFileMacOS))
	if err != nil {
		return errs.Wrap(err, "Could not write launch file")
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
	path, err := launchFilePath()
	if err != nil {
		return errs.Wrap(err, "Could not get launch file")
	}
	return os.Remove(path)
}

func (a *App) IsEnabled() (bool, error) {
	path, err := launchFilePath()
	if err != nil {
		return false, errs.Wrap(err, "Could not get launch file")
	}
	return fileutils.FileExists(path), nil
}

func launchFilePath() (string, error) {
	dir, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(dir, "Library/LaunchAgents", launchFileMacOS), nil
}
