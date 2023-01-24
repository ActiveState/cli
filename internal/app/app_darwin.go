package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/strutils"
)

const (
	execFileSource       = "exec.sh.tpl"
	launchFileSource     = "com.activestate.platform.app.plist.tpl"
	launchFileFormatName = "com.activestate.platform.%s.plist"
	autostartFileSource  = "com.activestate.platform.autostart.plist.tpl"
	iconFile             = "icon.icns"
)

var targetDirs = []string{
	"/Contents",
	"/Contents/MacOS",
	"/Contents/Resources",
}

func (a *App) install() error {
	// Create all of the necessary directories and files in a temporary directory
	// Then move the temporary directory to the final location which for macOS will be the Applications directory
	tmpDir, err := ioutil.TempDir("", fmt.Sprintf("%s-", a.Name))
	if err != nil {
		return errs.Wrap(err, "Could not create temporary directory")
	}
	defer os.RemoveAll(tmpDir)

	tmpAppPath := filepath.Join(tmpDir, fmt.Sprintf("%s.app", a.Name))
	if err := fileutils.Mkdir(tmpAppPath); err != nil {
		return errs.Wrap(err, "Could not create .app directory")
	}

	for _, t := range targetDirs {
		path := filepath.Join(tmpAppPath, t)
		err = fileutils.Mkdir(path)
		if err != nil {
			return errs.Wrap(err, "Could not create directory at %s", path)
		}
	}

	icon, err := assets.ReadFileBytes(a.options.IconFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	err = fileutils.WriteFile(filepath.Join(tmpAppPath, "Contents", "Resources", iconFile), icon)
	if err != nil {
		return errs.Wrap(err, "Could not write icon file")
	}

	err = a.createExecFile(filepath.Join(tmpAppPath, "Contents", "MacOS"))
	if err != nil {
		return errs.Wrap(err, "Could not create exec file")
	}

	err = a.createInfoFile(filepath.Join(tmpAppPath, "Contents"))
	if err != nil {
		return errs.Wrap(err, "Could not create info file")
	}

	installDir, err := installation.ApplicationInstallPath()
	if err != nil {
		return errs.Wrap(err, "Could not get installation path")
	}

	err = fileutils.MoveAllFiles(tmpDir, installDir)
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

	scriptFile := fmt.Sprintf("%s.sh", filepath.Base(a.Exec))

	content, err := strutils.ParseTemplate(
		string(asset),
		map[string]interface{}{
			"Exec": a.Exec,
			"Args": strings.Join(a.Args, " "),
		})
	if err != nil {
		return errs.Wrap(err, "Could not parse launch file source")
	}

	err = fileutils.WriteFile(filepath.Join(path, scriptFile), []byte(content))
	if err != nil {
		return errs.Wrap(err, "Could not write Info.plist file")
	}

	err = os.Chmod(filepath.Join(path, scriptFile), 0755)
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

	scriptFile := fmt.Sprintf("%s.sh", filepath.Base(a.Exec))

	content, err := strutils.ParseTemplate(
		string(asset),
		map[string]interface{}{
			"Exec":         scriptFile,
			"Interactive":  a.options.MacInteractive,
			"Icon":         a.options.IconFileName,
			"HideDockIcon": a.options.MacHideDockIcon,
			"IsGUIApp":     a.options.IsGUIApp,
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

	installPath, err := a.installPath()
	if err != nil {
		return errs.Wrap(err, "Could not get install path")
	}

	asset, err := assets.ReadFileBytes(autostartFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	content, err := strutils.ParseTemplate(
		string(asset),
		map[string]interface{}{
			"Label":       a.options.MacLabel,
			"Exec":        installPath,
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
	enabled, err := a.isAutostartEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}

	if !enabled {
		logging.Debug("Autostart is already disabled for %s", a.Name)
		return nil
	}
	path, err := a.autostartInstallPath()
	if err != nil {
		return errs.Wrap(err, "Could not get launch file")
	}
	return os.Remove(path)
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

func (a *App) installPath() (string, error) {
	dir, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	path := filepath.Join(dir, "Applications", fmt.Sprintf("%s.app", a.Name))
	return path, nil
}
