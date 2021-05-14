package main

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func autoUpdate(args []string, cfg *config.Instance, pjPath string) (bool, error) {
	disableAutoUpdate := strings.ToLower(os.Getenv(constants.DisableUpdates)) == "true"
	disableAutoUpdateCauseCI := (os.Getenv("CI") != "" || os.Getenv("BUILDER_OUTPUT") != "") && strings.ToLower(os.Getenv(constants.DisableUpdates)) != "false"
	updateIsRunning := funk.Contains(args, "update")
	testsAreRunning := condition.InTest()

	if testsAreRunning || updateIsRunning || disableAutoUpdate || disableAutoUpdateCauseCI || !osExeOverDayOld() {
		logging.Debug("Not running auto updates")
		return false, nil
	}

	// skip if version is locked
	// Todo: This may change
	if versionInfo, _ := projectfile.ParseVersionInfo(pjPath); versionInfo != nil {
		return false, nil
	}

	seconds := 1
	if secondsOverride := os.Getenv(constants.AutoUpdateTimeoutEnvVarName); secondsOverride != "" {
		override, err := strconv.Atoi(secondsOverride)
		if err == nil {
			seconds = override
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	defer cancel()
	m, err := model.NewSvcModel(ctx, cfg)
	if err != nil {
		return false, errs.Wrap(err, "Failed to create state-svc model")
	}
	up, err := m.InitiateDeferredUpdate("", "")
	if err != nil {
		return false, errs.Wrap(err, "Failed schedule the update.")
	}
	return up.Channel != "" && up.Version != "", nil
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

func osExeOverDayOld() bool {
	exe, err := os.Executable()
	if err != nil {
		logging.Error("Could not grab executable, error: %v", err)
		return false
	}
	return exeOverDayOld(exe)
}

func exeOverDayOld(exe string) bool {
	stat, err := os.Stat(exe)
	if err != nil {
		logging.Error("Could not stat file: %s, error: %v", exe)
		return false
	}
	diff := time.Now().Sub(stat.ModTime())
	return diff > 24*time.Hour
}
