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

func enable(params Params) error {
	enabled, err := isEnabled(params.Exec, params.options.LaunchFileName)
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}
	if enabled {
		return nil
	}

	if onDesktop() {
		// The user is installing while in a desktop environment. Install an autostart shortcut file.
		return enableOnDesktop(params)
	}
	// Probably in a server environment. Install to the user's ~/.profile.
	return enableOnServer(params)
}

func onDesktop() bool {
	return os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("DISPLAY") != ""
}

func enableOnDesktop(params Params) error {
	dir, err := prependHomeDir(autostartDir)
	if err != nil {
		return errs.Wrap(err, "Could not find autostart directory")
	}
	path := filepath.Join(dir, params.options.LaunchFileName)

	iconsDir, err := prependHomeDir(constants.IconsDir)
	if err != nil {
		return errs.Wrap(err, "")
	}
	iconsPath := filepath.Join(iconsDir, params.options.IconFileName)

	iconData, err := assets.ReadFileBytes(params.options.IconFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	scutOpts := shortcut.SaveOpts{
		Name:        params.Name,
		GenericName: params.options.GenericName,
		Comment:     params.options.Comment,
		Keywords:    params.options.Keywords,
		IconData:    iconData,
		IconPath:    iconsPath,
	}
	if _, err := shortcut.Save(params.Exec, path, params.Args, scutOpts); err != nil {
		return errs.Wrap(err, "Could not save autostart shortcut")
	}

	return nil
}

func enableOnServer(params Params) error {
	profile, err := prependHomeDir(autostartFile)
	if err != nil {
		return errs.Wrap(err, "Could not find ~/.profile")
	}

	esc := osutils.NewBashEscaper()
	exec := esc.Quote(params.Exec)
	for _, arg := range params.Args {
		exec += " " + esc.Quote(arg)
	}

	// Some older versions of the State Tool used a different ID for the autostart entry.
	err = sscommon.CleanRcFile(profile, sscommon.InstallID)
	if err != nil {
		return errs.Wrap(err, "Could not clean old autostart entry from %s", profile)
	}
	return sscommon.WriteRcData(exec, profile, sscommon.AutostartID)
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

func disable(params Params) error {
	enabled, err := isEnabled(params.Exec, params.options.LaunchFileName)
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}
	if !enabled {
		return nil
	}

	path, err := autostartPath(params.options.LaunchFileName)
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
	// Some older versions of the State Tool used a different ID for the autostart entry.
	if fileutils.FileExists(profile) {
		return sscommon.CleanRcFile(profile, sscommon.InstallID)
	}
	if fileutils.FileExists(profile) {
		return sscommon.CleanRcFile(profile, sscommon.AutostartID)
	}

	return nil
}

func isEnabled(params Params) (bool, error) {
	// Check for desktop autostart shortcut file.
	dir, err := prependHomeDir(autostartDir)
	if err != nil {
		return false, errs.Wrap(err, "Could not find autostart directory")
	}
	path := filepath.Join(dir, params.options.LaunchFileName)
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
		return strings.Contains(string(data), params.Exec), nil
	}

	return false, nil
}

func autostartPath(name string) (string, error) {
	dir, err := prependHomeDir(autostartDir)
	if err != nil {
		return "", errs.Wrap(err, "Could not find autostart directory")
	}
	path := filepath.Join(dir, name)

	if fileutils.FileExists(path) {
		return path, nil
	}

	// If on server, do not report ~/.profile as installed, as it would be removed on uninstall.
	return "", nil
}
