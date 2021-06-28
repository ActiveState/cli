package globaldefault

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/executor"
)

type DefaultConfigurer interface {
	sscommon.Configurable
	CachePath() string
}

// BinDir returns the global binary directory
func BinDir(cfg DefaultConfigurer) string {
	return filepath.Join(cfg.CachePath(), "bin")
}

func Prepare(cfg DefaultConfigurer, subshell subshell.SubShell) error {
	logging.Debug("Preparing globaldefault")
	binDir := BinDir(cfg)

	isWindowsAdmin, err := osutils.IsWindowsAdmin()
	if err != nil {
		logging.Error("Failed to determine if we are running as administrator: %v", err)
	}
	if isWindowsAdmin {
		logging.Debug("Skip preparation step as it is not supported for Windows Administrators.")
		return nil
	}
	if isOnPATH(binDir) {
		logging.Debug("Skip preparation step as it has been done previously for the current user.")
		return nil
	}

	if err := fileutils.MkdirUnlessExists(binDir); err != nil {
		return locale.WrapError(err, "err_globaldefault_bin_dir", "Could not create bin directory.")
	}

	envUpdates := map[string]string{
		"PATH": binDir,
	}

	if err := subshell.WriteUserEnv(cfg, envUpdates, sscommon.DefaultID, true); err != nil {
		return locale.WrapError(err, "err_globaldefault_update_env", "Could not write to user environment.")
	}

	return nil
}


// SetupDefaultActivation sets symlinks in the global bin directory to the currently activated runtime
func SetupDefaultActivation(subshell subshell.SubShell, cfg DefaultConfigurer, runtime *runtime.Runtime, projectPath string) error {
	logging.Debug("Setting up globaldefault")
	if err := Prepare(cfg, subshell); err != nil {
		return locale.WrapError(err, "err_globaldefault_prepare", "Could not prepare environment.")
	}

	exes, err := runtime.ExecutablePaths()
	if err != nil {
		return locale.WrapError(err, "err_globaldefault_rtexes", "Could not retrieve runtime executables")
	}

	fw := executor.NewWithBinPath(projectPath, BinDir(cfg))
	if err := fw.Update(exes); err != nil {
		return locale.WrapError(err, "err_globaldefault_fw", "Could not set up forwarders")
	}

	if err := cfg.Set(constants.GlobalDefaultPrefname, projectPath); err != nil {
		return locale.WrapError(err, "err_set_default_config", "Could not set default project in config file")
	}

	return nil
}
