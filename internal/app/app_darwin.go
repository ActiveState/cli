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
	"github.com/ActiveState/cli/internal/strutils"
)

const (
	execFileSource   = "exec.sh.tpl"
	launchFileSource = "com.activestate.platform.app.plist.tpl"
	iconFile         = "icon.icns"
	assetAppDir      = "placeholder.app"
)

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

	appDir, err := assets.OpenFile(assetAppDir)
	if err != nil {
		return errs.Wrap(err, "Could not read app directory asset")
	}

	info, err := appDir.Stat()
	if err != nil {
		return errs.Wrap(err, "Could not stat app directory asset")
	}

	err = generateAppDir(tmpAppPath, info.Name())
	if err != nil {
		return errs.Wrap(err, "Could not generate app directory")
	}

	if err := a.createIcon(tmpAppPath); err != nil {
		return errs.Wrap(err, "Could not create icon")
	}

	if err := a.createExecFile(tmpAppPath); err != nil {
		return errs.Wrap(err, "Could not create exec file")
	}

	if err := a.createInfoFile(tmpAppPath); err != nil {
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

func generateAppDir(createPath, assetName string) error {
	listing, err := assets.ReadDir(assetName)
	if err != nil {
		return errs.Wrap(err, "Could not read app directory asset")
	}

	for _, entry := range listing {
		if !entry.IsDir() {
			continue
		}

		err := fileutils.Mkdir(createPath, entry.Name())
		if err != nil {
			return errs.Wrap(err, "Could not create directory")
		}

		err = generateAppDir(filepath.Join(createPath, entry.Name()), filepath.Join(assetName, entry.Name()))
		if err != nil {
			return errs.Wrap(err, "Could not generate app directory")
		}
	}

	return nil
}

func (a *App) createIcon(path string) error {
	icon, err := assets.ReadFileBytes(a.options.IconFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	if err = fileutils.WriteFile(filepath.Join(path, "Contents", "Resources", iconFile), icon); err != nil {
		return errs.Wrap(err, "Could not write icon file")
	}

	return nil
}

func (a *App) createExecFile(base string) error {
	path := filepath.Join(base, "Contents", "MacOS")
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

func (a *App) createInfoFile(base string) error {
	path := filepath.Join(base, "Contents")
	asset, err := assets.ReadFileBytes(launchFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	scriptFile := fmt.Sprintf("%s.sh", filepath.Base(a.Exec))

	content, err := strutils.ParseTemplate(
		string(asset),
		map[string]interface{}{
			"Exec":         scriptFile,
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
