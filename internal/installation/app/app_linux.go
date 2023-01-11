package app

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

func (a *App) install() error {
	return nil
}

func (a *App) uninstall() error {
	return nil
}

func (a *App) enableAutostart() error {
	enabled, err := a.isAutostartEnabled()
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

func (a *App) onDesktop() bool {
	return os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("DISPLAY") != ""
}

func (a *App) enableOnDesktop() error {
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

func (a *App) enableOnServer() error {
	profile, err := prependHomeDir(autostartFile)
	if err != nil {
		return errs.Wrap(err, "Could not find ~/.profile")
	}

	esc := osutils.NewBashEscaper()
	exec := esc.Quote(a.Exec)
	for _, arg := range a.Args {
		exec += " " + esc.Quote(arg)
	}

	// Some older versions of the State Tool used a different ID for the autostart entry.
	err = sscommon.CleanRcFile(profile, sscommon.InstallID)
	if err != nil {
		return errs.Wrap(err, "Could not clean old autostart entry from %s", profile)
	}
	return sscommon.WriteRcData(exec, profile, sscommon.InstallID)
}

func (a *App) disableAutostart() error {
	enabled, err := a.isAutostartEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}
	if !enabled {
		return nil
	}

	path, err := a.installPath()
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

func (a *App) isAutostartEnabled() (bool, error) {
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

func (a *App) autostartInstallPath() (string, error) {
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

func prependHomeDir(path string) (string, error) {
	homeDir, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	if testDir, ok := os.LookupEnv(constants.AutostartPathOverrideEnvVarName); ok {
		homeDir = testDir
	}
	return filepath.Join(homeDir, path), nil
}
