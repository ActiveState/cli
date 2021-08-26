package main

import (
	"os"
	"os/exec"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/internal/updater"
)

func autoUpdate(args []string, cfg *config.Instance, out output.Outputer, svcm *svcmanager.Manager, pjPath string) (bool, error) {
	disableAutoUpdate := strings.ToLower(os.Getenv(constants.DisableUpdates)) == "true"
	disableAutoUpdateCauseCI := (os.Getenv("CI") != "" || os.Getenv("BUILDER_OUTPUT") != "") && strings.ToLower(os.Getenv(constants.DisableUpdates)) != "false"
	updateIsRunning := funk.Contains(args, "update")
	testsAreRunning := condition.InTest()

	if testsAreRunning || updateIsRunning || disableAutoUpdate || disableAutoUpdateCauseCI || !osExeOverDayOld() {
		logging.Debug("Not running auto updates")
		return false, nil
	}

	updated, resultVersion := updater.AutoUpdate(svcm, cfg, pjPath, out)
	if !updated {
		return false, nil
	}

	out.Notice(output.Heading(locale.Tl("auto_update_title", "Auto Update")))
	out.Notice(locale.Tr("auto_update_to_version", constants.Version, resultVersion))
	code, err := relaunch()
	if err != nil {
		return true, errs.WrapExitCode(err, code)
	}
	return true, nil
}

// When an update was found and applied, re-launch the update with the current
// arguments and wait for return before exitting.
func relaunch() (int, error) {
	cmd := exec.Command(appinfo.StateApp().Exec(), os.Args[1:]...)
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

func osExeOverDayOld() bool {
	exe, err := os.Executable()
	if err != nil {
		logging.Error("Could not grab executable, error: %v", err)
		return false
	}
	return exeOverDayOld(exe)
}
