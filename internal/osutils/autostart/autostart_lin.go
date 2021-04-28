//+build linux

package autostart

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils/shortcut"
	"github.com/gobuffalo/packr"
	"github.com/mitchellh/go-homedir"
)

const (
	autostartDir = ".config/autostart"
)

func (a *App) Enable() error {
	enabled, err := a.IsEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}
	if enabled {
		return nil
	}

	dir, err := prependHomeDir(autostartDir)
	if err != nil {
		return errs.Wrap(err, "Could not find autostart directory")
	}
	path := filepath.Join(dir, constants.TrayLaunchFileName)

	scut, err := shortcut.New(appinfo.TrayApp().Exec(), path)
	if err != nil {
		return errs.Wrap(err, "Could not construct autostart shortcut")
	}

	iconsDir, err := prependHomeDir(constants.IconsDir)
	if err != nil {
		return errs.Wrap(err, "")
	}
	iconsPath := filepath.Join(iconsDir, constants.TrayIconFileName)

	box := packr.NewBox("../../../assets")
	iconData := box.Bytes(constants.TrayIconFileSource)

	scutOpts := shortcut.ShortcutSaveOpts{
		GenericName: constants.TrayGenericName,
		Comment:     constants.TrayComment,
		Keywords:    constants.TrayKeywords,
		IconData:    iconData,
		IconPath:    iconsPath,
	}
	if err := scut.Save(constants.TrayAppName, scutOpts); err != nil {
		return errs.Wrap(err, "Could not save autostart shortcut")
	}

	return nil
}

func (a *App) Disable() error {
	enabled, err := a.IsEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}
	if !enabled {
		return nil
	}

	dir, err := prependHomeDir(autostartDir)
	if err != nil {
		return errs.Wrap(err, "Could not find autostart directory")
	}
	path := filepath.Join(dir, constants.TrayLaunchFileName)

	return os.Remove(path)
}

func (a *App) IsEnabled() (bool, error) {
	dir, err := prependHomeDir(autostartDir)
	if err != nil {
		return false, errs.Wrap(err, "Could not find autostart directory")
	}
	path := filepath.Join(dir, constants.TrayLaunchFileName)

	return fileutils.FileExists(path), nil
}

func prependHomeDir(path string) (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(homeDir, path), nil
}
