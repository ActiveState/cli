package autostart

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
)

const (
	autostartFile = ".profile"
)

func (a *App) enable() error {
	if err := legacyDisableOnDesktop(a.options.LaunchFileName); err != nil {
		logging.Error("Cannot properly disable autostart (desktop): %v", err)
	}

	isEnabled, err := a.IsEnabled()
	if err != nil {
		return err
	}
	if isEnabled {
		return nil
	}

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
	return sscommon.WriteRcData(exec, profile, sscommon.AutostartID)
}

func (a *App) disable() error {
	if err := legacyDisableOnDesktop(a.options.LaunchFileName); err != nil {
		return err
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

// isEnabled does not verify legacy "Desktop" autostart setups, so should not be
// used apart from Enable/Disable, and should be used only within tests.
func (a *App) isEnabled() (bool, error) {
	// check for ~/.profile modification.
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

func (a *App) installPath() (string, error) {
	// do not report ~/.profile as installed, as it would be removed on uninstall.
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

// https://activestatef.atlassian.net/browse/DX-1677
func legacyDisableOnDesktop(launchFileName string) error {
	dir, err := prependHomeDir(".config/autostart")
	if err != nil {
		return errs.Wrap(err, "Could not find autostart directory")
	}

	path := filepath.Join(dir, launchFileName)

	if fileutils.FileExists(path) {
		err := os.Remove(path)
		if err != nil {
			return errs.Wrap(err, "Could not remove autostart shortcut")
		}
	}

	return nil
}
