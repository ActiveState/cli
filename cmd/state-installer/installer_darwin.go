//go:build darwin
// +build darwin

package main

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
)

// installLauncher installs files in the /Application directory
func (i *Installer) installLauncher() error {
	sourcePath := filepath.Join(i.sourcePath, "system")
	if !fileutils.DirExists(sourcePath) {
		multilog.Error("Installation does not have a system path")
		return nil
	}

	// Detect the launcher path
	launcherPath, err := installation.LauncherInstallPath()
	if err != nil {
		return errs.Wrap(err, "Could not get system install path")
	}

	err = os.RemoveAll(filepath.Join(launcherPath, constants.MacOSApplicationName))
	if err != nil {
		return errs.Wrap(err, "Could not remove old app directory")
	}

	// ensure launcherPath exists
	err = fileutils.MkdirUnlessExists(launcherPath)
	if err != nil {
		return errs.Wrap(err, "Application directory %s did not exist, and failed to create it", launcherPath)
	}

	err = fileutils.MoveAllFilesCrossDisk(sourcePath, launcherPath)
	if err != nil {
		return errs.Wrap(err, "Could not create application directory")
	}

	trayExec, err := installation.NewExecInDir(i.path, installation.TrayExec)
	if err != nil {
		return locale.WrapError(err, "err_tray_info")
	}

	exeName := filepath.Base(trayExec)
	toTray := filepath.Join(launcherPath, constants.MacOSApplicationName, "Contents", "MacOS", exeName)
	err = createNewSymlink(trayExec, toTray)
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

func (i *Installer) PrepareBinTargets() error {
	return nil
}
