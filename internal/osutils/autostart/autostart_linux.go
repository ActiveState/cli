package autostart

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
)

func enable(exec string, opts Options) error {
	profile, err := profilePath()
	if err != nil {
		return errs.Wrap(err, "Could not get profile path")
	}

	enabled, err := isEnabled(exec, opts)
	if err != nil {
		return errs.Wrap(err, "Could not check if autostart is already enabled")
	}
	if enabled {
		return nil
	}

	esc := osutils.NewBashEscaper()
	exec = esc.Quote(exec)
	for _, arg := range opts.Args {
		exec += " " + esc.Quote(arg)
	}

	if err := sscommon.WriteRcData(exec, profile, sscommon.AutostartID); err != nil {
		return errs.Wrap(err, "Could not update %s with autostart entry", profile)
	}

	return nil
}

func disable(exec string, opts Options) error {
	profile, err := profilePath()
	if err != nil {
		return errs.Wrap(err, "Could not get profile path")
	}

	if fileutils.FileExists(profile) {
		if err := sscommon.CleanRcFile(profile, sscommon.AutostartID); err != nil {
			return errs.Wrap(err, "Could not clean autostart entry from %s", profile)
		}
	}

	return nil
}

// isEnabled, for Linux, does not verify legacy "Desktop" autostart setups, so
// should be used carefully with that in mind. External code should only use it
// within tests.
func isEnabled(exec string, opts Options) (bool, error) {
	profile, err := profilePath()
	if err != nil {
		return false, errs.Wrap(err, "Could not get profile path")
	}

	if fileutils.FileExists(profile) {
		data, err := fileutils.ReadFile(profile)
		if err != nil {
			return false, errs.Wrap(err, "Could not read %s", profile)
		}
		return strings.Contains(string(data), exec), nil
	}

	return false, nil
}

func autostartPath(name string, opts Options) (string, error) {
	// Linux uses ~/.profile modification for autostart, there is no actual
	// autostartPath.
	return "", nil
}

func upgrade(_ string, opts Options) error {
	if err := legacyDisableOnDesktop(opts.LaunchFileName); err != nil {
		return errs.Wrap(err, "Could not disable legacy autostart (desktop)")
	}

	profile, err := profilePath()
	if err != nil {
		return errs.Wrap(err, "Could not get profile path")
	}

	if err := legacyRemoveAutostartEntry(profile); err != nil {
		return errs.Wrap(err, "Could not clean up legacy autostart entry (server)")
	}

	return nil
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

func profilePath() (string, error) {
	autostartFile := ".profile"

	profile, err := prependHomeDir(autostartFile)
	if err != nil {
		return "", errs.Wrap(err, "Could not find ~/%s", autostartFile)
	}

	return profile, nil
}

// https://activestatef.atlassian.net/browse/DX-1677
func legacyDisableOnDesktop(launchFileName string) error {
	autostartDir := ".config/autostart"

	dir, err := prependHomeDir(autostartDir)
	if err != nil {
		return errs.Wrap(err, "Could not find ~/%s", autostartDir)
	}

	path := filepath.Join(dir, launchFileName)

	if fileutils.FileExists(path) {
		err := os.Remove(path)
		if err != nil {
			return errs.Wrap(err, "Could not remove shortcut")
		}
	}

	return nil
}

// https://activestatef.atlassian.net/browse/DX-1677
func legacyRemoveAutostartEntry(profileFile string) error {
	if !fileutils.FileExists(profileFile) {
		return nil
	}

	if err := sscommon.CleanRcFile(profileFile, sscommon.InstallID); err != nil {
		return errs.Wrap(err, "Could not clean %s", profileFile)
	}

	return nil
}
