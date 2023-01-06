package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/strutils"
)

const (
	execFileSource       = "exec.sh.tpl"
	launchFileSource     = "com.activestate.platform.state.plist.tpl"
	launchFileLabel      = "com.activestate.state-svc"
	launchFileFormatName = "com.activestate.platform.%s.plist"
)

type target struct {
	path string
	dir  bool
}

var targets = []target{
	{"/Contents", true},
	{"/Contents/MacOS", true},
	{"/Contents/Resources", true},
	{"/Contents/MacOS/.placeholder", false},
}

func (a *App) install() error {
	// Create all of the necessary directories and files in a temporary directory
	// Then move the temporary directory to the final location which for macOS will be the Applications directory
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s.app", a.Name))
	err := fileutils.Mkdir(tmpPath)
	if err != nil {
		return errs.Wrap(err, "Could not create .app directory")
	}

	for _, t := range targets {
		path := filepath.Join(tmpPath, t.path)
		if t.dir {
			err = fileutils.Mkdir(path)
			if err != nil {
				return errs.Wrap(err, "Could not create directory at %s", path)
			}
		} else {
			err = fileutils.Touch(path)
			if err != nil {
				return errs.Wrap(err, "Could not create file at %s", path)
			}
		}
	}

	err = a.createExecFile(filepath.Join(tmpPath, "Contents", "MacOS"))
	if err != nil {
		return errs.Wrap(err, "Could not create exec file")
	}

	err = a.createInfoFile(filepath.Join(tmpPath, "Contents"))
	if err != nil {
		return errs.Wrap(err, "Could not create info file")
	}

	// TODO: Rename icon file
	icon, err := assets.ReadFileBytes("state-tray.icns")
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	err = fileutils.WriteFile(filepath.Join(tmpPath, "Contents", "Resources", "icon.icns"), icon)
	if err != nil {
		return errs.Wrap(err, "Could not write icon file")
	}

	dir, err := user.HomeDir()
	if err != nil {
		return errs.Wrap(err, "Could not get home directory")
	}

	err = fileutils.MoveAllFiles(tmpPath, filepath.Join(dir, "/Applications"))
	if err != nil {
		return errs.Wrap(err, "Could not move .app to Applications directory")
	}

	return nil
}

func (a *App) createExecFile(path string) error {
	asset, err := assets.ReadFileBytes(execFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	content, err := strutils.ParseTemplate(
		string(asset),
		map[string]interface{}{
			"Exec": a.Exec,
			"Args": strings.Join(a.Args, " "),
		})
	if err != nil {
		return errs.Wrap(err, "Could not parse launch file source")
	}

	err = fileutils.WriteFile(filepath.Join(path, fmt.Sprintf("%s.sh", a.Name)), []byte(content))
	if err != nil {
		return errs.Wrap(err, "Could not write Info.plist file")
	}

	err = os.Chmod(filepath.Join(path, fmt.Sprintf("%s.sh", a.Name)), 0755)
	if err != nil {
		return errs.Wrap(err, "Could not make executable")
	}

	return nil
}

func (a *App) createInfoFile(path string) error {
	asset, err := assets.ReadFileBytes(launchFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	content, err := strutils.ParseTemplate(
		string(asset),
		map[string]interface{}{
			"Exec":        a.Exec,
			"Args":        strings.Join(a.Args, " "),
			"Interactive": true,
		})
	if err != nil {
		return errs.Wrap(err, "Could not parse launch file source")
	}

	err = fileutils.WriteFile(filepath.Join(path, "Info.plist"), []byte(content))
	if err != nil {
		return errs.Wrap(err, "Could not write Info.plist file")
	}

	return nil
}

func (a *App) uninstall() error {
	installDir := filepath.Join("/Applications", fmt.Sprintf("%s.app", a.Name))
	if !fileutils.DirExists(installDir) {
		return nil
	}

	err := os.RemoveAll(installDir)
	if err != nil {
		return errs.Wrap(err, "Could not remove .app from Applications directory")
	}

	return nil
}

func (a *App) enableAutostart() error {
	enabled, err := a.IsAutostartEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}

	if enabled {
		return nil
	}

	path, err := a.AutostartInstallPath()
	if err != nil {
		return errs.Wrap(err, "Could not get launch file")
	}

	asset, err := assets.ReadFileBytes(launchFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	content, err := strutils.ParseTemplate(
		string(asset),
		map[string]interface{}{
			"Label":       a.options.MacLabel,
			"Exec":        a.Exec,
			"Args":        strings.Join(a.Args, " "),
			"Interactive": a.options.MacInteractive,
		})
	if err != nil {
		return errs.Wrap(err, "Could not parse %s", fmt.Sprintf(launchFileFormatName, filepath.Base(a.Exec)))
	}

	if err = fileutils.WriteFile(path, []byte(content)); err != nil {
		return errs.Wrap(err, "Could not write launch file")
	}
	return nil
}

func (a *App) disableAutostart() error {
	return errs.New("Not implemented")
}

func (a *App) isAutostartEnabled() (bool, error) {
	path, err := a.AutostartInstallPath()
	if err != nil {
		return false, errs.Wrap(err, "Could not get launch file")
	}
	return fileutils.FileExists(path), nil
}

func (a *App) autostartInstallPath() (string, error) {
	dir, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	path := filepath.Join(dir, "Library/LaunchAgents", fmt.Sprintf(launchFileFormatName, filepath.Base(a.Exec)))
	return path, nil
}
