package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils/shortcut"
)

var startupPath = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "Startup")

func (a *App) install() error {
	return nil
}

func (a *App) uninstall() error {
	return nil
}

func (a *App) enableAutostart() error {
	enabled, err := a.isAutostartEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app is enabled")
	}
	if enabled {
		return nil
	}

	name := formattedName(a.Name)
	s := shortcut.New(startupPath, name, a.Exec, a.Args...)
	if err := s.Enable(); err != nil {
		return errs.Wrap(err, "Could not create shortcut")
	}

	icon, err := assets.ReadFileBytes(a.options.IconFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	err = s.SetIconBlob(icon)
	if err != nil {
		return errs.Wrap(err, "Could not set icon for shortcut file")
	}

	err = s.SetWindowStyle(shortcut.Minimized)
	if err != nil {
		return errs.Wrap(err, "Could not set shortcut to minimized")
	}

	return nil
}

func (a *App) disableAutostart() error {
	enabled, err := a.isAutostartEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}

	if !enabled {
		return nil
	}
	return os.Remove(a.shortcutFilename())
}

func (a *App) isAutostartEnabled() (bool, error) {
	return fileutils.FileExists(a.shortcutFilename()), nil
}

func (a *App) autostartInstallPath() (string, error) {
	return a.shortcutFilename(), nil
}

func (a *App) shortcutFilename() string {
	name := formattedName(a.Name)
	return filepath.Join(startupPath, name+".lnk")
}

func formattedName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}
