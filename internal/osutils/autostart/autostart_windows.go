package autostart

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils/shortcut"
)

var startupPath = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "Startup")

func enable(exec string, opts Options) error {
	enabled, err := isEnabled(exec, opts)
	if err != nil {
		return errs.Wrap(err, "Could not check if app is enabled")
	}
	if enabled {
		return nil
	}

	name := formattedName(opts.Name)
	s := shortcut.New(startupPath, name, exec, opts.Args...)
	if err := s.Enable(); err != nil {
		return errs.Wrap(err, "Could not create shortcut")
	}

	icon, err := assets.ReadFileBytes(opts.IconFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	err = s.SetIconBlob(icon)
	if err != nil {
		return errs.Wrap(err, "Could not set icon for shortcut file")
	}
	s.SetWindowStyle(shortcut.Minimized)

	return nil
}

func disable(exec string, opts Options) error {
	enabled, err := isEnabled(exec, opts)
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}

	if !enabled {
		return nil
	}
	return os.Remove(shortcutFilename(opts.Name))
}

func isEnabled(_ string, opts Options) (bool, error) {
	return fileutils.FileExists(shortcutFilename(opts.Name)), nil
}

func autostartPath(_ string, opts Options) (string, error) {
	return shortcutFilename(opts.Name), nil
}

func upgrade(exec string, opts Options) error {
	return nil
}

func shortcutFilename(name string) string {
	name = formattedName(name)
	if testDir, ok := os.LookupEnv(constants.AutostartPathOverrideEnvVarName); ok {
		startupPath = testDir
	}
	return filepath.Join(startupPath, name+".lnk")
}

func formattedName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}
