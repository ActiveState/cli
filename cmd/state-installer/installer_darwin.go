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
	"github.com/ActiveState/cli/internal/multilog"
)

// installLauncher installs files in the /Application directory
func (i *Installer) installLauncher() error {
	sourcePath := filepath.Join(i.payloadPath, "system")
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

	return nil
}

func (i *Installer) PrepareBinTargets() error {
	return nil
}
