package main

import (
	"os"
	"os/exec"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/updater"
)

func autoUpdate(args []string, out output.Outputer, pjPath string) (int, error) {
	disableAutoUpdate := strings.ToLower(os.Getenv(constants.DisableUpdates)) == "true"
	disableAutoUpdateCauseCI := (os.Getenv("CI") != "" || os.Getenv("BUILDER_OUTPUT") != "") && strings.ToLower(os.Getenv(constants.DisableUpdates)) != "false"
	updateIsRunning := funk.Contains(args, "update")
	testsAreRunning := condition.InTest()

	if testsAreRunning || updateIsRunning || disableAutoUpdate || disableAutoUpdateCauseCI {
		return 0, nil
	}

	updated, resultVersion := updater.AutoUpdate(pjPath)
	if !updated {
		return 0, nil
	}

	out.Notice(locale.Tr("auto_update_to_version", constants.Version, resultVersion))
	return relaunch()
}

// When an update was found and applied, re-launch the update with the current
// arguments and wait for return before exitting.
func relaunch() (int, error) {
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	logging.Debug("Running command: %s", strings.Join(cmd.Args, " "))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Start()
	if err != nil {
		logging.Error("Failed to start command: %v", err)
	}

	err = cmd.Wait()
	if err != nil {
		logging.Error("relaunched cmd returned error: %v", err)
	}

	return osutils.CmdExitCode(cmd), err
}
