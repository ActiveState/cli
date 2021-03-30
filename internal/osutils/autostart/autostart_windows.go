package autostart

import (
	"os"
	"path/filepath"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils/shortcut"
)

var startupPath = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "Startup")

func (a *App) Enable() error {
	if a.IsEnabled() {
		return nil
	}
	s, err := shortcut.New(startupPath, a.Name, a.Exec)
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
	if !a.IsEnabled() {
		return nil
	}
	return os.Remove(a.shortcutFilename())
}

func (a *App) IsEnabled() bool {
	return fileutils.FileExists(a.shortcutFilename())
}

func (a *App) shortcutFilename() string {
	return filepath.Join(startupPath, a.Name+".lnk")
}
