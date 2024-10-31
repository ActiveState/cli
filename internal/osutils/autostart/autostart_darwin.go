package autostart

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/strutils"
)

const (
	launchFileFormatName = "com.activestate.platform.%s.plist"
	autostartFileSource  = "com.activestate.platform.autostart.plist.tpl"
)

func enable(exec string, opts Options) error {
	enabled, err := isEnabled(exec, opts)
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}

	if enabled {
		return nil
	}

	path, err := autostartPath(exec, opts)
	if err != nil {
		return errs.Wrap(err, "Could not get launch file")
	}

	asset, err := assets.ReadFileBytes(autostartFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	content, err := strutils.ParseTemplate(
		string(asset),
		map[string]interface{}{
			"Label":       opts.MacLabel,
			"Exec":        exec,
			"Interactive": opts.MacInteractive,
		}, nil)
	if err != nil {
		return errs.Wrap(err, "Could not parse %s", fmt.Sprintf(launchFileFormatName, filepath.Base(exec)))
	}

	if err = fileutils.WriteFile(path, []byte(content)); err != nil {
		return errs.Wrap(err, "Could not write launch file")
	}

	return nil
}

func disable(exec string, opts Options) error {
	enabled, err := isEnabled(exec, opts)
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}

	if !enabled {
		logging.Debug("Autostart is already disabled for %s", opts.Name)
		return nil
	}

	path, err := autostartPath(exec, opts)
	if err != nil {
		return errs.Wrap(err, "Could not get launch file")
	}

	return os.Remove(path)
}

func isEnabled(exec string, opts Options) (bool, error) {
	path, err := autostartPath(exec, opts)
	if err != nil {
		return false, errs.Wrap(err, "Could not get launch file")
	}

	return fileutils.FileExists(path), nil
}

func autostartPath(_ string, opts Options) (string, error) {
	dir, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	if testDir, ok := os.LookupEnv(constants.AutostartPathOverrideEnvVarName); ok {
		dir = testDir
	}
	path := filepath.Join(dir, "Library/LaunchAgents", fmt.Sprintf(launchFileFormatName, opts.LaunchFileName))
	return path, nil
}

func upgrade(exec string, opts Options) error {
	path, err := autostartPath(exec, opts)
	if err != nil {
		return errs.Wrap(err, "Could not get launch file")
	}

	legacy, err := isLegacyPlist(path)
	if err != nil {
		return errs.Wrap(err, "Could not check if legacy plist")
	}

	if !legacy {
		return nil
	}

	logging.Debug("Legacy autostart file found, removing: %s", path)
	return os.Remove(path)
}
