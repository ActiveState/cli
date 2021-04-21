// +build darwin

package installer

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/gobuffalo/packr"
)

const (
	appDir  = "/Applications"
	appName = "ActiveState Desktop.app"
)

// InstallSystemFiles installs files in the /Application directory
func InstallSystemFiles(installDir string) error {
	err := os.RemoveAll(filepath.Join(appDir, appName))
	if err != nil {
		return errs.Wrap(err, "Could not remove old app directory")
	}

	box := packr.NewBox("../../../../assets/macOS/app")
	err = box.Walk(func(path string, _ packr.File) error {
		if fileutils.IsDir(path) {
			err := fileutils.Mkdir(filepath.Join(appDir, path))
			if err != nil {
				return errs.Wrap(err, "Could not create directory")
			}
		} else {
			err := fileutils.WriteFile(filepath.Join(appDir, path), box.Bytes(path))
			if err != nil {
				return errs.Wrap(err, "Could not write icon file")
			}
		}
		return nil
	})
	if err != nil {
		return errs.Wrap(err, "Could not create application directory")
	}

	trayInfo := appinfo.TrayApp()
	err = createNewSymlink(filepath.Join(installDir, filepath.Base(trayInfo.Exec())), filepath.Join(appDir, appName, "Contents", "MacOS", filepath.Base(trayInfo.Exec())))
	if err != nil {
		return errs.Wrap(err, "Could not create state-tray symlink")
	}

	return nil
}

func createNewSymlink(target, filename string) error {
	if fileutils.FileExists(filename) {
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
