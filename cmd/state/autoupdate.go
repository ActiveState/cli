package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/thoas/go-funk"
)

type ErrStateExe struct{ *locale.LocalizedError }

type ErrExecuteRelaunch struct{ *errs.WrapperError }

const CfgKeyLastCheck = "auto_update_lastcheck"

func init() {
	configMediator.RegisterOption(constants.AutoUpdateConfigKey, configMediator.Bool, configMediator.EmptyEvent, configMediator.EmptyEvent)
}

func autoUpdate(args []string, cfg *config.Instance, an analytics.Dispatcher, out output.Outputer) (bool, error) {
	profile.Measure("autoUpdate", time.Now())

	defer func() {
		if err := cfg.Set(CfgKeyLastCheck, time.Now()); err != nil {
			multilog.Error("Failed to store last update check: %s", errs.JoinMessage(err))
		}
	}()

	if !shouldRunAutoUpdate(args, cfg, an) {
		return false, nil
	}

	// Check for available update
	checker := updater.NewDefaultChecker(cfg, an)
	up, err := checker.Check()
	if err != nil {
		return false, errs.Wrap(err, "Failed to check for update")
	}
	if up == nil {
		logging.Debug("No update found")
		return false, nil
	}

	if !isEnabled(cfg) {
		logging.Debug("Not performing autoupdates because user turned off autoupdates.")
		an.EventWithLabel(anaConst.CatUpdates, anaConst.ActShouldUpdate, anaConst.UpdateLabelDisabledConfig)
		out.Notice(output.Title(locale.Tl("update_available_header", "Auto Update")))
		out.Notice(locale.Tr("update_available", constants.Version, up.Version))
		return false, nil
	}

	out.Notice(output.Title(locale.Tl("auto_update_title", "Auto Update")))
	out.Notice(locale.Tr("auto_update_to_version", constants.Version, up.Version))

	logging.Debug("Auto updating to %s", up.Version)

	err = up.InstallBlocking("")
	if err != nil {
		if os.IsPermission(err) {
			an.EventWithLabel(anaConst.CatUpdates, anaConst.ActUpdateInstall, anaConst.UpdateLabelFailed, &dimensions.Values{
				Version: ptr.To(up.Version),
				Error:   ptr.To("Could not update the state tool due to insufficient permissions."),
			})
			return false, locale.WrapInputError(err, locale.Tl("auto_update_permission_err", "", constants.DocumentationURL, errs.JoinMessage(err)))
		}
		if errs.Matches(err, &updater.ErrorInProgress{}) {
			an.EventWithLabel(anaConst.CatUpdates, anaConst.ActUpdateInstall, anaConst.UpdateLabelFailed, &dimensions.Values{
				Version: ptr.To(up.Version),
				Error:   ptr.To(anaConst.UpdateErrorInProgress),
			})
			return false, nil
		}
		an.EventWithLabel(anaConst.CatUpdates, anaConst.ActUpdateInstall, anaConst.UpdateLabelFailed, &dimensions.Values{
			Version: ptr.To(up.Version),
			Error:   ptr.To(anaConst.UpdateErrorInstallFailed),
		})
		return false, locale.WrapError(err, locale.T("auto_update_failed"))
	}

	out.Notice(locale.Tr("auto_update_relaunch"))
	out.Notice("") // Ensure output doesn't stick to our messaging

	code, err := relaunch(args)
	if err != nil {
		var msg string
		if errs.Matches(err, &ErrStateExe{}) {
			msg = anaConst.UpdateErrorExecutable
		} else if errs.Matches(err, &ErrExecuteRelaunch{}) {
			msg = anaConst.UpdateErrorRelaunch
		}
		an.EventWithLabel(anaConst.CatUpdates, anaConst.ActUpdateRelaunch, anaConst.UpdateLabelFailed, &dimensions.Values{
			Version: ptr.To(up.Version),
			Error:   ptr.To(msg),
		})
		return true, errs.Silence(errs.WrapExitCode(err, code))
	}

	an.EventWithLabel(anaConst.CatUpdates, anaConst.ActUpdateRelaunch, anaConst.UpdateLabelSuccess, &dimensions.Values{
		Version: ptr.To(up.Version),
	})
	return true, nil
}

func isEnabled(cfg *config.Instance) bool {
	if !cfg.IsSet(constants.AutoUpdateConfigKey) {
		return !condition.IsLTS()
	}
	return cfg.GetBool(constants.AutoUpdateConfigKey)
}

func shouldRunAutoUpdate(args []string, cfg *config.Instance, an analytics.Dispatcher) bool {
	var (
		shouldUpdate bool
		label        string
	)

	switch {
	// In a forward
	case os.Getenv(constants.ForwardedStateEnvVarName) == "true":
		logging.Debug("Not running auto updates because we're in a forward")
		shouldUpdate = false
		label = anaConst.UpdateLabelForward

	// Forced enabled (breaks out of switch)
	case os.Getenv(constants.TestAutoUpdateEnvVarName) == "true":
		logging.Debug("Forcing auto update as it was forced by env var")
		shouldUpdate = true
		label = anaConst.UpdateLabelTrue

	// In unit test
	case condition.InUnitTest():
		logging.Debug("Not running auto updates in unit tests")
		shouldUpdate = false
		label = anaConst.UpdateLabelUnitTest

	// Running command that could conflict
	case funk.Contains(args, "update") || funk.Contains(args, "export") || funk.Contains(args, "_prepare") || funk.Contains(args, "clean"):
		logging.Debug("Not running auto updates because current command might conflict")
		shouldUpdate = false
		label = anaConst.UpdateLabelConflict

	// Updates are disabled
	case strings.ToLower(os.Getenv(constants.DisableUpdates)) == "true":
		logging.Debug("Not running auto updates because updates are disabled by env var")
		shouldUpdate = false
		label = anaConst.UpdateLabelDisabledEnv

	// We're on CI
	case (condition.OnCI()) && strings.ToLower(os.Getenv(constants.DisableUpdates)) != "false":
		logging.Debug("Not running auto updates because we're on CI")
		shouldUpdate = false
		label = anaConst.UpdateLabelCI

	// Exe is not old enough
	case isFreshInstall():
		logging.Debug("Not running auto updates because we just freshly installed")
		shouldUpdate = false
		label = anaConst.UpdateLabelFreshInstall

	// Already checked less than 60 minutes ago
	case time.Now().Sub(cfg.GetTime(CfgKeyLastCheck)).Minutes() < float64(60):
		logging.Debug("Not running auto update because we already checked it less than 60 minutes ago")
		shouldUpdate = false
		label = anaConst.UpdateLabelTooFreq

	case cfg.GetString(updater.CfgKeyInstallVersion) != "":
		logging.Debug("Not running auto update because a specific version had been installed on purpose")
		shouldUpdate = false
		label = anaConst.UpdateLabelLocked
	}

	an.EventWithLabel(anaConst.CatUpdates, anaConst.ActShouldUpdate, label)
	return shouldUpdate
}

// When an update was found and applied, re-launch the update with the current
// arguments and wait for return before exitting.
func relaunch(args []string) (int, error) {
	exec, err := installation.StateExec()
	if err != nil {
		return -1, &ErrStateExe{locale.WrapError(err, "err_state_exec")}
	}

	code, _, err := exeutils.ExecuteAndPipeStd(exec, args[1:], []string{fmt.Sprintf("%s=true", constants.ForwardedStateEnvVarName)})
	if err != nil {
		return code, &ErrExecuteRelaunch{errs.Wrap(err, "Forwarded command after auto-updating failed. Exit code: %d", code)}
	}

	return code, nil
}

func isFreshInstall() bool {
	exe := osutils.Executable()
	stat, err := os.Stat(exe)
	if err != nil {
		multilog.Error("Could not stat file: %s, error: %v", exe, err)
		return true
	}
	diff := time.Now().Sub(stat.ModTime())
	return diff < 24*time.Hour
}
