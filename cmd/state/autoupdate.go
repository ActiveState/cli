package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

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

func autoUpdate(args []string, cfg *config.Instance, out output.Outputer) (bool, error) {
	disableAutoUpdate := strings.ToLower(os.Getenv(constants.DisableUpdates)) == "true"
	disableAutoUpdateCauseCI := (condition.OnCI()) && strings.ToLower(os.Getenv(constants.DisableUpdates)) != "false"
	updateIsRunning := funk.Contains(args, "update")
	testsAreRunning := condition.InTest()

	if testsAreRunning || updateIsRunning || disableAutoUpdate || disableAutoUpdateCauseCI || !osExeOverDayOld() {
		logging.Debug("Not running auto updates")
		return false, nil
	}

	// Check for available update
	checker := updater.NewDefaultChecker(cfg)
	up, err := checker.Check()
	if err != nil {
		return false, errs.Wrap(err, "Failed to check for update")
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
		if os.IsPermission(errs.InnerError(err)) {
			return false, locale.WrapInputError(err, "auto_update_permission_err")
		}
		if errors.As(err, new(*lockfile.AlreadyLockedError)) {
			log("Auto update failed because the update lock file is already in use")
			return false, nil
		}
		return false, errs.Wrap(err, "Failed to install update")
	}

	out.Notice(locale.Tr("auto_update_relaunch"))
	code, err := relaunch()
	if err != nil {
		return true, errs.WrapExitCode(err, code)
	}

	return true, nil
}

// When an update was found and applied, re-launch the update with the current
// arguments and wait for return before exitting.
func relaunch() (int, error) {
	logging.Debug("Running command: %s", strings.Join(os.Args[1:], " "))
	code, _, err := exeutils.ExecuteAndPipeStd(appinfo.StateApp().Exec(), os.Args[1:], []string{})
	if err != nil {
		return code, locale.WrapError(err, "err_autoupdate_relaunch_wait", "Could not forward your command after auto-updating, please manually run your command again.")
	}

	return code, nil
}

func osExeOverDayOld() bool {
	exe, err := os.Executable()
	if err != nil {
		logging.Error("Could not grab executable, error: %v", err)
		return false
	}
	return exeOverDayOld(exe)
}
