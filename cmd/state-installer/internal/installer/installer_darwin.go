// +build darwin

package installer

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/gobuffalo/packr"
)

const (
	contentsDir  = "/Applications/state-tray.app/Contents"
	macOSDir     = "MacOS"
	resourcesDir = "Resources"
)

// InstallSystemFiles installs files in the /Application directory
func InstallSystemFiles(installDir string) error {
	err := createAppDirStructure()
	if err != nil {
		return errs.Wrap(err, "Could not create app directory structure")
	}

	err = populateAppDir(installDir)
	if err != nil {
		return errs.Wrap(err, "Could not populate state-tray app directory")
	}

	return nil
}

func createAppDirStructure() error {
	err := fileutils.Mkdir(contentsDir)
	if err != nil {
		return errs.Wrap(err, "Could not create Contents app dir")
	}

	err = fileutils.Mkdir(filepath.Join(contentsDir, macOSDir))
	if err != nil {
		return errs.Wrap(err, "Could not create MacOS app dir")
	}

	err = fileutils.Mkdir(filepath.Join(contentsDir, resourcesDir))
	if err != nil {
		return errs.Wrap(err, "Could not create Resources app dir")
	}

	return nil
}

func populateAppDir(installDir string) error {
	box := packr.NewBox("../../../../assets/macOS")
	err := fileutils.WriteFile(filepath.Join(contentsDir, resourcesDir), box.Bytes("state-tray.icns"))
	if err != nil {
		return errs.Wrap(err, "Could not write icon file")
	}

	err = fileutils.WriteFile(contentsDir, box.Bytes("Info.plist"))
	if err != nil {
		return errs.Wrap(err, "Could not write info plist file")
	}

	err = os.Symlink(filepath.Join(installDir, "state-tray"), filepath.Join(contentsDir, macOSDir, "state-tray"))
	if err != nil {
		return errs.Wrap(err, "Could not create state-tray symlink")
	}

	err = os.Symlink(filepath.Join(installDir, "state-svc"), filepath.Join(contentsDir, macOSDir, "state-svc"))
	if err != nil {
		return errs.Wrap(err, "Could not create state-svc symlink")
	}

	return nil
}
