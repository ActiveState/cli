//+build linux

package autostart

import (
	"os"
	"path"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/gobuffalo/packr"
	"github.com/mitchellh/go-homedir"
)

func (a *App) Enable() error {
	enabled, err := a.IsEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}
	if enabled {
		return nil
	}

	filePath, err := launchFilePath()
	if err != nil {
		return errs.Wrap(err, "Could not get launch file")
	}
	fileName := path.Base(filePath)

	box := packr.NewBox("../../../assets")
	err = fileutils.WriteFile(filePath, box.Bytes(fileName))
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

const (
	autostartDir     = ".config/autostart"
	launchFileUbuntu = "state-tray.desktop"
)

func launchFilePath() (string, error) {
	dir, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(dir, autostartDir, launchFileUbuntu), nil
}
