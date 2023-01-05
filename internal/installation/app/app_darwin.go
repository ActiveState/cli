package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/strutils"
)

const (
	launchFileSource = "com.activestate.platform.state.plist.tpl"
	launchFileLabel  = "com.activestate.state-svc"
)

func (a *App) Install() error {
	// Create all of the necessary directories and files in a temporary directory
	// Then move the temporary directory to the final location which for macOS will be the Applications directory
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s.app", a.Name))
	err := fileutils.Mkdir(tmpPath)
	if err != nil {
		return errs.Wrap(err, "Could not create .app directory")
	}

	err = fileutils.Mkdir(filepath.Join(tmpPath, "Contents"))
	if err != nil {
		return errs.Wrap(err, "Could not create Contents directory")
	}

	err = fileutils.Mkdir(filepath.Join(tmpPath, "Contents", "MacOS"))
	if err != nil {
		return errs.Wrap(err, "Could not create MacOS directory")
	}

	err = fileutils.Mkdir(filepath.Join(tmpPath, "Contents", "Resources"))
	if err != nil {
		return errs.Wrap(err, "Could not create Resources directory")
	}

	err = fileutils.Touch(filepath.Join(tmpPath, "Contents", "MacOS", ".placeholder"))
	if err != nil {
		return errs.Wrap(err, "Could not create .placeholder file")
	}

	asset, err := assets.ReadFileBytes(launchFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	content, err := strutils.ParseTemplate(
		string(asset),
		map[string]interface{}{
			"Label":       launchFileLabel,
			"Exec":        a.Exec,
			"Args":        strings.Join(a.Args, " "),
			"Interactive": true,
		})
	if err != nil {
		return errs.Wrap(err, "Could not parse launch file source")
	}

	err = fileutils.WriteFile(filepath.Join(tmpPath, "Contents", "Info.plist"), []byte(content))
	if err != nil {
		return errs.Wrap(err, "Could not write Info.plist file")
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

	return errs.New("Not implemented")
}

func (a *App) Uninstall() error {
	// Remove the application from the Applications directory
	return errs.New("Not implemented")
}

func (a *App) EnableAutostart() error {
	return errs.New("Not implemented")
}

func (a *App) DisableAutostart() error {
	return errs.New("Not implemented")
}
