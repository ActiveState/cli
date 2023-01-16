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

func enable(params Params) error {
	enabled, err := IsEnabled(params)
	if err != nil {
		return errs.Wrap(err, "Could not check if app is enabled")
	}
	if enabled {
		return nil
	}

	name := formattedName(params.Name)
	s := shortcut.New(startupPath, name, params.Exec, params.Args...)
	if err := s.Enable(); err != nil {
		return errs.Wrap(err, "Could not create shortcut")
	}

	icon, err := assets.ReadFileBytes(params.options.IconFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	err = s.SetIconBlob(icon)
	if err != nil {
		return errs.Wrap(err, "Could not set icon for shortcut file")
	}

	err = s.SetWindowStyle(shortcut.Minimized)
	if err != nil {
		return errs.Wrap(err, "Could not set shortcut to minimized")
	}

	return nil
}

func disable(params Params) error {
	enabled, err := isEnabled(params)
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}

	if !enabled {
		return nil
	}
	return os.Remove(shortcutFilename(params.Name))
}

func isEnabled(params Params) (bool, error) {
	return fileutils.FileExists(shortcutFilename(params.Name)), nil
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
