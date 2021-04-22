//+build linux

package autostart

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/gobuffalo/packr"
	"github.com/mitchellh/go-homedir"
)

func (a *App) Enable() error {
	appDir, err := dirPath(applicationDir)
	if err != nil {
		return errs.Wrap(err, "Could not get application file")
	}
	appPath := filepath.Join(appDir, launchFile)

	enabled, err := a.IsEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}
	if enabled {
		return nil
	}

	dir, err := dirPath(autostartDir)
	if err != nil {
		return errs.Wrap(err, "Could not get autostart file")
	}
	path := filepath.Join(dir, launchFile)

	if err = os.Symlink(appPath, path); err != nil {
		return errs.Wrap(err, "Could not create symlink")
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

	dir, err := dirPath(autostartDir)
	if err != nil {
		return errs.Wrap(err, "Could not get autostart file")
	}
	path := filepath.Join(dir, launchFile)

	return os.Remove(path)
}

func (a *App) IsEnabled() (bool, error) {
	appDir, err := dirPath(applicationDir)
	if err != nil {
		return false, errs.Wrap(err, "Could not get application file")
	}
	appPath := filepath.Join(appDir, launchFile)

	if !fileutils.FileExists(appPath) {
		if err = setupAppFile(appPath); err != nil {
			return false, errs.Wrap(err, "Could not setup app file")
		}
	}

	dir, err := dirPath(autostartDir)
	if err != nil {
		return false, errs.Wrap(err, "Could not get autostart file")
	}
	path := filepath.Join(dir, launchFile)

	return fileutils.FileExists(path), nil
}

const (
	applicationDir = ".local/share/applications"
	autostartDir   = ".config/autostart"
	iconsDir       = ".local/share/icons/hicolor/scalable/apps"
	launchFile     = "state-tray.desktop"
	iconFileState  = "state-tray.svg"
	iconFileBase   = "icon.svg"
)

func dirPath(dir string) (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(homeDir, dir), nil
}

func setupAppFile(appPath string) error {
	t := template.New("")
	t, err := t.Parse(desktopFileTmpl)
	if err != nil {
		return errs.Wrap(err, "Could not parse desktop file template")
	}

	buf := &bytes.Buffer{}
	data := desktopFileData{Exec: appinfo.TrayApp().Exec()}
	if err = t.Execute(buf, data); err != nil {
		return errs.Wrap(err, "Could not execute template")
	}

	icons, err := dirPath(iconsDir)
	if err != nil {
		return errs.Wrap(err, "Could not get icons directory")
	}
	iconPath := filepath.Join(icons, iconFileState)

	box := packr.NewBox("../../../assets")
	if err := fileutils.WriteFile(iconPath, box.Bytes(iconFileBase)); err != nil {
		return errs.Wrap(err, "Could not write icon file")
	}

	if err := fileutils.WriteFile(appPath, buf.Bytes()); err != nil {
		return errs.Wrap(err, "Could not write application file")
	}

	file, err := os.Open(appPath)
	if err != nil {
		return errs.Wrap(err, "Could not open application file")
	}
	err = file.Chmod(0770)
	file.Close()
	if err != nil {
		return errs.Wrap(err, "Could not make file executable")
	}

	// set the executable as trusted so users do not need to do it manually
	// gio is "Gnome input/output"
	cmd := exec.Command("gio", "set", appPath, "metadata::trusted", "true")
	if err := cmd.Run(); err != nil {
		return errs.Wrap(err, "Could not set application file as trusted")
	}

	return nil
}

type desktopFileData struct {
	Exec string
}

var desktopFileTmpl = `
[Desktop Entry]
Name=ActiveState Desktop
GenericName=Language Runtime Manager
Type=Application
Comment=Manage ActiveState Platform Projects
Exec="{{ .Exec }}"
Terminal=false
Keywords=activestate;state;tray;language;runtime;python;perl;tcl;
Categories=Utility;Development;
Hidden=false
NoDisplay=false
StartupNotify=false
Icon=state-tray
Name[en_US]=ActiveState Desktop
`
