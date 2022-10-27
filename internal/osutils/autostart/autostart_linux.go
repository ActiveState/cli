package autostart

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/shortcut"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
)

const (
	autostartDir  = ".config/autostart"
	autostartFile = ".profile"
)

func (a *app) enable() error {
	enabled, err := a.IsEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}
	if enabled {
		return nil
	}

	if a.onDesktop() {
		// The user is installing while in a desktop environment. Install an autostart shortcut file.
		return a.enableOnDesktop()
	}
	// Probably in a server environment. Install to the user's ~/.profile.
	return a.enableOnServer()
}

func (a *app) onDesktop() bool {
	return os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("DISPLAY") != ""
}

func (a *app) enableOnDesktop() error {
	dir, err := prependHomeDir(autostartDir)
	if err != nil {
		return errs.Wrap(err, "Could not find autostart directory")
	}
	path := filepath.Join(dir, a.options.LaunchFileName)

	iconsDir, err := prependHomeDir(constants.IconsDir)
	if err != nil {
		return errs.Wrap(err, "")
	}
	iconsPath := filepath.Join(iconsDir, a.options.IconFileName)

	iconData, err := assets.ReadFileBytes(a.options.IconFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	scutOpts := shortcut.SaveOpts{
		Name:        a.Name,
		GenericName: a.options.GenericName,
		Comment:     a.options.Comment,
		Keywords:    a.options.Keywords,
		IconData:    iconData,
		IconPath:    iconsPath,
	}
	if _, err := shortcut.Save(a.Exec, path, a.Args, scutOpts); err != nil {
		return errs.Wrap(err, "Could not save autostart shortcut")
	}

	return nil
}

func (a *app) enableOnServer() error {
	profile, err := prependHomeDir(autostartFile)
	if err != nil {
		return errs.Wrap(err, "Could not find ~/.profile")
	}

	esc := osutils.NewBashEscaper()
	exec := esc.Quote(a.Exec)
	for _, arg := range a.Args {
		exec += " " + esc.Quote(arg)
	}

	return sscommon.WriteRcData(exec, profile, sscommon.InstallID)
}

// Path returns the path to the installed autostart shortcut file.
// The installer keeps track of this file for later removal on uninstall.
func (a *app) InstallPath() (string, error) {
	dir, err := prependHomeDir(autostartDir)
	if err != nil {
		return "", errs.Wrap(err, "Could not find autostart directory")
	}
	path := filepath.Join(dir, a.options.LaunchFileName)

	if fileutils.FileExists(path) {
		return path, nil
	}

	// If on server, do not report ~/.profile as installed, as it would be removed on uninstall.
	return "", nil
}

func (a *app) disable() error {
	enabled, err := a.IsEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}
	if !enabled {
		return nil
	}

	path, err := a.InstallPath()
	if err != nil {
		return err
	}

	// Remove the desktop autostart shortcut file if it's there.
	if fileutils.FileExists(path) {
		err := os.Remove(path)
		if err != nil {
			return errs.Wrap(err, "Could not remove autostart shortcut")
		}
	}

	// Remove the ~/.profile modification if it's there.
	profile, err := prependHomeDir(autostartFile)
	if err != nil {
		return errs.Wrap(err, "Could not find ~/.profile")
	}
	if fileutils.FileExists(profile) {
		return sscommon.CleanRcFile(profile, sscommon.InstallID)
	}

	return nil
}

func (a *app) IsEnabled() (bool, error) {
	// Check for desktop autostart shortcut file.
	dir, err := prependHomeDir(autostartDir)
	if err != nil {
		return false, errs.Wrap(err, "Could not find autostart directory")
	}
	path := filepath.Join(dir, a.options.LaunchFileName)
	if fileutils.FileExists(path) {
		return true, nil
	}

	// Or check for ~/.profile modification.
	profile, err := prependHomeDir(autostartFile)
	if err != nil {
		return false, errs.Wrap(err, "Could not find ~/.profile")
	}
	if fileutils.FileExists(profile) {
		data, err := fileutils.ReadFile(profile)
		if err != nil {
			return false, errs.Wrap(err, "Could not read ~/.profile")
		}
		return strings.Contains(string(data), a.Exec), nil
	}

	return false, nil
}

func prependHomeDir(path string) (string, error) {
	homeDir, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(homeDir, path), nil
}
