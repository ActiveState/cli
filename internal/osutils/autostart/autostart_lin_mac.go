// +build !windows

package autostart

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/gobuffalo/packr"
	"github.com/mitchellh/go-homedir"
)

const (
	relativeLaunchPath = "Library/LaunchAgents"
	launchFile         = "com.activestate.platform.state-tray.plist"
)

func (a *App) Enable() error {
	if a.IsEnabled() {
		return nil
	}

	path, err := launchFilePath()
	if err != nil {
		return err
	}

	box := packr.NewBox("../../../assets")
	err = fileutils.WriteFile(path, box.Bytes(launchFile))
	if err != nil {
		return errs.Wrap(err, "Could not write launch file")
	}
	return nil
}

func (a *App) Disable() error {
	if !a.IsEnabled() {
		return nil
	}
	path, err := launchFilePath()
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func (a *App) IsEnabled() bool {
	path, err := launchFilePath()
	if err != nil {
		logging.Error("Could not get launch file: %v", err)
	}
	return fileutils.FileExists(path)
}

func launchFilePath() (string, error) {
	dir, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(dir, relativeLaunchPath, launchFile), nil
}
