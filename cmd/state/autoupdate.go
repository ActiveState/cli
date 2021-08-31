package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/lockfile"
	"github.com/gofrs/flock"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/updater"
)

const CfgKeyLastCheck = "auto_update_lastcheck"

func autoUpdate(args []string, cfg *config.Instance, out output.Outputer) (bool, error) {
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

	// Protect against multiple updates happening simultaneously
	fileLock := flock.New(filepath.Join(cfg.ConfigPath(), "install.lock"))
	lockSuccess, err := fileLock.TryLock()
	if err != nil {
		return false, errs.Wrap(err, "Could not create file lock required to install update")
	}
	if !lockSuccess {
		logging.Debug("Another update is already in progress")
		return false, nil
	}
	defer fileLock.Unlock()

	targetDir := filepath.Dir(appinfo.StateApp().Exec())
	err = up.InstallBlocking(targetDir)
	if err != nil {
		log := logging.Error
		innerErr := errs.InnerError(err)
		if os.IsPermission(innerErr) {
			return false, locale.WrapInputError(err, "auto_update_permission_err", innerErr.Error())
		}
		if errors.As(err, new(*lockfile.AlreadyLockedError)) {
			log("Auto update failed because the update lock file is already in use")
			return false, nil
		}
		return false, errs.Wrap(err, "Failed to install update")
	}

	out.Notice(locale.Tr("auto_update_relaunch"))
	out.Notice("") // Ensure output doesn't stick to our messaging
	code, err := relaunch(args)
	if err != nil {
		return true, errs.WrapExitCode(err, code)
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

	// Already running manual update
	case funk.Contains(args, "update"):
		logging.Debug("Not running auto updates because 'update' in args")
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
	}

	return true
}

// When an update was found and applied, re-launch the update with the current
// arguments and wait for return before exitting.
func relaunch(args []string) (int, error) {
	code, _, err := exeutils.ExecuteAndPipeStd(appinfo.StateApp().Exec(), args[1:], []string{fmt.Sprintf("%s=true", constants.ForwardedStateEnvVarName)})
	if err != nil {
		return code, locale.WrapError(err, "err_autoupdate_relaunch_wait", "Could not forward your command after auto-updating, please manually run your command again.")
	}

	return code, nil
}

func isFreshInstall() bool {
	exe, err := osutils.Executable()
	if err != nil {
		logging.Error("Could not grab executable, error: %v", err)
		return true
	}
	stat, err := os.Stat(exe)
	if err != nil {
		logging.Error("Could not stat file: %s, error: %v", exe)
		return true
	}
	diff := time.Now().Sub(stat.ModTime())
	return diff < 24*time.Hour
}
