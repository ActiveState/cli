package globaldefault

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/runbits/runtime/target"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/pkg/platform/runtime/executors"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/runtime"
)

type DefaultConfigurer interface {
	sscommon.Configurable
}

// BinDir returns the global binary directory
func BinDir() string {
	return storage.GlobalBinDir()
}

func Prepare(cfg DefaultConfigurer, shell subshell.SubShell) error {
	logging.Debug("Preparing globaldefault")
	binDir := BinDir()

	isWindowsAdmin, err := osutils.IsAdmin()
	if err != nil {
		multilog.Error("Failed to determine if we are running as administrator: %v", err)
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

	// Configure available shells
	err = subshell.ConfigureAvailableShells(shell, cfg, envUpdates, sscommon.DefaultID, true)
	if err != nil {
		return locale.WrapError(err, "err_globaldefault_update_env")
	}

	return nil
}

// SetupDefaultActivation sets symlinks in the global bin directory to the currently activated runtime
func SetupDefaultActivation(subshell subshell.SubShell, cfg DefaultConfigurer, runtime *runtime.Runtime, proj *project.Project) error {
	logging.Debug("Setting up globaldefault")
	if err := Prepare(cfg, subshell); err != nil {
		return locale.WrapError(err, "err_globaldefault_prepare", "Could not prepare environment.")
	}

	env := runtime.Env()
	exes, err := osutils.ExecutablePaths(env.Variables)
	if err != nil {
		return errs.Wrap(err, "Could not get executable paths")
	}

	target := target.NewProjectTargetCache(proj, storage.GlobalBinDir(), nil, target.TriggerActivate)
	execInit := executors.New(BinDir())
	if err := execInit.Apply(svcctl.NewIPCSockPathFromGlobals().String(), target, env.Variables, exes); err != nil {
		return locale.WrapError(err, "err_globaldefault_fw", "Could not set up forwarders")
	}

	projectDir := filepath.Dir(proj.Source().Path())
	if err := cfg.Set(constants.GlobalDefaultPrefname, projectDir); err != nil {
		return locale.WrapError(err, "err_set_default_config", "Could not update config file with your project")
	}

	return nil
}

func IsSet(cfg DefaultConfigurer) bool {
	return cfg.GetString(constants.GlobalDefaultPrefname) != ""
}

func ResetDefaultActivation(shell subshell.SubShell, cfg DefaultConfigurer) (bool, error) {
	logging.Debug("Resetting globaldefault")

	projectDir := cfg.GetString(constants.GlobalDefaultPrefname)
	if projectDir == "" {
		logging.Debug("No global project is set.")
		return false, nil // nothing to reset
	}

	execInit := executors.New(BinDir())
	if err := execInit.Clean(); err != nil {
		return false, locale.WrapError(err, "err_globaldefault_fw_cleanup", "Could not clean up forwarders")
	}

	envUpdates := map[string]string{}

	// Configure available shells
	err := subshell.ConfigureAvailableShells(shell, cfg, envUpdates, sscommon.DefaultID, true)
	if err != nil {
		return false, locale.WrapError(err, "err_globaldefault_update_env")
	}

	err = cfg.Set(constants.GlobalDefaultPrefname, "")
	if err != nil {
		return false, locale.WrapError(err, "err_reset_default_config", "Could not remove your project from config file")
	}

	return true, nil
}
