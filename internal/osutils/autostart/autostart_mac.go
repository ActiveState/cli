//go:build darwin
// +build darwin

package autostart

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/mitchellh/go-homedir"
)

var data = map[AppName]options{
	Tray: {
		launchFileName: "com.activestate.platform.state-tray.plist",
	},
	Service: {
		launchFileName: "com.activestate.platform.state-svc.plist",
	},
}

func (a *App) enable() error {
	enabled, err := a.IsEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}

	if enabled {
		return nil
	}

	path, err := a.Path()
	if err != nil {
		return errs.Wrap(err, "Could not get launch file")
	}

	launchFile, err := assets.ReadFileBytes(a.options.launchFileName)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}
	err = fileutils.WriteFile(path, launchFile)
	if err != nil {
		return errs.Wrap(err, "Could not write launch file")
	}
	return nil
}

func (a *App) disable() error {
	enabled, err := a.IsEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}

	if !enabled {
		return nil
	}
	path, err := a.Path()
	if err != nil {
		return errs.Wrap(err, "Could not get launch file")
	}
	return os.Remove(path)
}

func (a *App) IsEnabled() (bool, error) {
	path, err := a.Path()
	if err != nil {
		return false, errs.Wrap(err, "Could not get launch file")
	}
	return fileutils.FileExists(path), nil
}

func (a *App) Path() (string, error) {
	dir, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(dir, "Library/LaunchAgents", a.options.launchFileName), nil
}
