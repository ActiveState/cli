package main

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/updater"
)

func autoUpdate(args []string, out output.Outputer, pjPath string) (bool, int, error) {
	disableAutoUpdate := strings.ToLower(os.Getenv(constants.DisableUpdates)) == "true"
	disableAutoUpdateCauseCI := (os.Getenv("CI") != "" || os.Getenv("BUILDER_OUTPUT") != "") && strings.ToLower(os.Getenv(constants.DisableUpdates)) != "false"
	updateIsRunning := funk.Contains(args, "update")
	testsAreRunning := condition.InTest()

	if testsAreRunning || updateIsRunning || disableAutoUpdate || disableAutoUpdateCauseCI || !exeOverDayOld() {
		logging.Debug("Not running auto updates")
		return false, 0, nil
	}

	updated, resultVersion := updater.AutoUpdate(pjPath, out)
	if !updated {
		return false, 0, nil
	}

	defer updater.Cleanup()

	out.Notice(locale.Tr("auto_update_to_version", constants.Version, resultVersion))
	code, err := relaunch()
	return true, code, err
}

// When an update was found and applied, re-launch the update with the current
// arguments and wait for return before exitting.
func relaunch() (int, error) {
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	logging.Debug("Running command: %s", strings.Join(cmd.Args, " "))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Start()
	if err != nil {
		return 1, locale.WrapError(err, "err_autoupdate_relaunch_start",
			"Could not start updated State Tool after auto-updating, please manually run your command again, if the problem persists please reinstall the State Tool.")
	}

	err = cmd.Wait()
	if err != nil {
		return osutils.CmdExitCode(cmd), locale.WrapError(err, "err_autoupdate_relaunch_wait", "Could not forward your command after auto-updating, please manually run your command again.")
	}

	return osutils.CmdExitCode(cmd), nil
}

func exeOverDayOld() bool {
	exe, err := os.Executable()
	if err != nil {
		logging.Error("Could not grab executable, error: %v", err)
		return false
	}
	stat, err := os.Stat(exe)
	if err != nil {
		logging.Error("Could not stat file: %s, error: %v", exe)
		return false
	}
	diff := time.Now().Sub(stat.ModTime())
	return diff > 24*time.Hour
}
