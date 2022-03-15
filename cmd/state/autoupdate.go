package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/updater"
)

const CfgKeyLastCheck = "auto_update_lastcheck"

type forwardExitError struct {
	code int
}

func (fe *forwardExitError) Error() string  { return "forwardExitError" }
func (fe *forwardExitError) Unwrap() error  { return nil }
func (fe *forwardExitError) IsSilent() bool { return true }
func (fe *forwardExitError) ExitCode() int  { return fe.code }

func autoUpdate(args []string, cfg *config.Instance, out output.Outputer) (bool, error) {
	profile.Measure("autoUpdate", time.Now())
	defer func() {
		if err := cfg.Set(CfgKeyLastCheck, time.Now()); err != nil {
			multilog.Error("Failed to store last update check: %s", errs.JoinMessage(err))
		}
	}()

	if !shouldRunAutoUpdate(args, cfg) {
		return false, nil
	}

	// Check for available update
	checker := updater.NewDefaultChecker(cfg)
	up, err := checker.Check()
	if err != nil {
		return false, errs.Wrap(err, "Failed to check for update")
	}
	if up == nil {
		logging.Debug("No update found")
		return false, nil
	}

	out.Notice(output.Heading(locale.Tl("auto_update_title", "Auto Update")))
	out.Notice(locale.Tr("auto_update_to_version", constants.Version, up.Version))

	logging.Debug("Auto updating to %s", up.Version)

	err = up.InstallBlocking("")
	if err != nil {
		innerErr := errs.InnerError(err)
		if os.IsPermission(innerErr) {
			return false, locale.WrapInputError(err, "auto_update_permission_err", "", constants.DocumentationURL, errs.JoinMessage(err))
		}
		if errs.Matches(err, &updater.ErrorInProgress{}) {
			logging.Debug("Update already in progress")
			return false, nil
		}
		return false, locale.WrapError(err, "auto_update_failed")
	}

	out.Notice(locale.Tr("auto_update_relaunch"))
	out.Notice("") // Ensure output doesn't stick to our messaging
	code := relaunch(args)
	if code != 0 {
		return true, &forwardExitError{code}
	}

	return true, nil
}

func shouldRunAutoUpdate(args []string, cfg *config.Instance) bool {
	switch {
	// In a forward
	case os.Getenv(constants.ForwardedStateEnvVarName) == "true":
		logging.Debug("Not running auto updates because we're in a forward")
		return false

	// Forced enabled (breaks out of switch)
	case os.Getenv(constants.TestAutoUpdateEnvVarName) == "true":
		logging.Debug("Forcing auto update as it was forced by env var")
		return true

	// In unit test
	case condition.InUnitTest():
		logging.Debug("Not running auto updates in unit tests")
		return false

	// Running command that could conflict
	case funk.Contains(args, "update") || funk.Contains(args, "export") || funk.Contains(args, "_prepare") || funk.Contains(args, "clean"):
		logging.Debug("Not running auto updates because current command might conflict")
		return false

	// Updates are disabled
	case strings.ToLower(os.Getenv(constants.DisableUpdates)) == "true":
		logging.Debug("Not running auto updates because updates are disabled by env var")
		return false

	// We're on CI
	case (condition.OnCI()) && strings.ToLower(os.Getenv(constants.DisableUpdates)) != "false":
		logging.Debug("Not running auto updates because we're on CI")
		return false

	// Exe is not old enough
	case isFreshInstall():
		logging.Debug("Not running auto updates because we just freshly installed")
		return false

	// Already checked less than 60 minutes ago
	case time.Now().Sub(cfg.GetTime(CfgKeyLastCheck)).Minutes() < float64(60):
		logging.Debug("Not running auto update because we already checked it less than 60 minutes ago")
		return false

	case cfg.GetString(updater.CfgKeyInstallVersion) != "":
		logging.Debug("Not running auto update because a specific version had been installed on purpose")
		return false
	}

	return true
}

// When an update was found and applied, re-launch the update with the current
// arguments and wait for return before exitting.
func relaunch(args []string) int {
	code, _, err := exeutils.ExecuteAndPipeStd(appinfo.StateApp().Exec(), args[1:], []string{fmt.Sprintf("%s=true", constants.ForwardedStateEnvVarName)})
	if err != nil {
		logging.Debug("Forwarded command after auto-updating failed. Exit code: %d", code)
		return code
	}

	return code
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
