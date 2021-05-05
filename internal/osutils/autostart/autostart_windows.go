package autostart

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils/shortcut"
)

var startupPath = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "Startup")

func (a *App) Enable() error {
	enabled, err := a.IsEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app is enabled")
	}
	if enabled {
		return nil
	}

	name := formattedName(a.Name)
	s, err := shortcut.New(startupPath, name, a.Exec)
	if err != nil {
		return errs.Wrap(err, "Could not create shortcut")
	}
	box := packr.NewBox("../../../assets")
	if err := s.SetIconBlob(box.Bytes("icon.ico")); err != nil {
		return errs.Wrap(err, "Could not set icon for shortcut file")
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
	return os.Remove(a.shortcutFilename())
}

func (a *App) IsEnabled() (bool, error) {
	return fileutils.FileExists(a.shortcutFilename()), nil
}

func (a *App) shortcutFilename() string {
	name := formattedName(a.Name)
	return filepath.Join(startupPath, name+".lnk")
}

func formattedName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}
