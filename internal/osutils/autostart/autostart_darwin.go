package autostart

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
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

	installPath, err := installPath(opts.Name)
	if err != nil {
		return errs.Wrap(err, "Could not get install path")
	}

	asset, err := assets.ReadFileBytes(autostartFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	content, err := strutils.ParseTemplate(
		string(asset),
		map[string]interface{}{
			"Label":       opts.MacLabel,
			"Exec":        installPath,
			"Interactive": opts.MacInteractive,
		})
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

func autostartPath(exec string, _ Options) (string, error) {
	dir, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	path := filepath.Join(dir, "Library/LaunchAgents", fmt.Sprintf(launchFileFormatName, filepath.Base(exec)))
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

func installPath(name string) (string, error) {
	dir, err := installation.ApplicationInstallPath()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	path := filepath.Join(dir, fmt.Sprintf("%s.app", name))
	return path, nil
}
