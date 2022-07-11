package autostart

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils/shortcut"
	"github.com/mitchellh/go-homedir"
)

const (
	autostartDir = ".config/autostart"
)

func (a *App) enable() error {
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
	path := filepath.Join(dir, a.options.launchFileName)

	iconsDir, err := prependHomeDir(constants.IconsDir)
	if err != nil {
		return errs.Wrap(err, "")
	}
	iconsPath := filepath.Join(iconsDir, a.options.iconFileName)

	iconData, err := assets.ReadFileBytes(a.options.iconFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	scutOpts := shortcut.SaveOpts{
		Name:        a.Name,
		GenericName: a.options.genericName,
		Comment:     a.options.comment,
		Keywords:    a.options.keywords,
		IconData:    iconData,
		IconPath:    iconsPath,
	}
	if _, err := shortcut.Save(a.Exec, path, scutOpts); err != nil {
		return errs.Wrap(err, "Could not save autostart shortcut")
	}

	return nil
}

func (a *App) Path() (string, error) {
	dir, err := prependHomeDir(autostartDir)
	if err != nil {
		return "", errs.Wrap(err, "Could not find autostart directory")
	}
	path := filepath.Join(dir, a.options.launchFileName)

	return path, nil
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
		return err
	}

	return os.Remove(path)
}

func (a *App) IsEnabled() (bool, error) {
	dir, err := prependHomeDir(autostartDir)
	if err != nil {
		return false, errs.Wrap(err, "Could not find autostart directory")
	}
	path := filepath.Join(dir, a.options.launchFileName)

	return fileutils.FileExists(path), nil
}

func prependHomeDir(path string) (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(homeDir, path), nil
}
