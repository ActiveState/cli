package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/process"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, errs.Join(err, ": ").Error())
		fmt.Fprintln(os.Stderr, "To retry run %s", strings.Join(os.Args, " "))
		os.Exit(1)
	}
}

func run() error {
	exe, err := osutils.Executable()
	if err != nil {
		return errs.Wrap(err, "Could not detect executable path")
	}

	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config")
	}

	var installPath string
	if len(os.Args) > 1 {
		installPath = os.Args[1]
	} else {
		installPath, err = installation.InstallPath()
		if err != nil {
			return errs.Wrap(err, "Retrieving installPath")
		}
	}

	svcInfo, err := appinfo.SvcApp()
	if err != nil {
		return errs.Wrap(err, "Could not detect svc application information")
	}

	trayInfo, err := appinfo.TrayApp()
	if err != nil {
		return errs.Wrap(err, "Could not detect tray application information")
	}

	// Stop state-svc before accessing its files
	if fileutils.FileExists(svcInfo.Exec()) {
		exitCode, _, err := exeutils.Execute(svcInfo.Exec(), []string{"stop"}, nil)
		if err != nil {
			return errs.Wrap(err, "Stopping %s returned error", svcInfo.Name())
		}
		if exitCode != 0 {
			return errs.New("Stopping %s exited with code %d", svcInfo.Name(), exitCode)
		}
	}

	// Todo: https://www.pivotaltracker.com/story/show/177585085
	// Yes this is awkward right now
	if err := stopTrayApp(cfg); err != nil {
		return errs.Wrap(err, "Failed to stop %s", trayInfo.Name())
	}

	// Todo: https://www.pivotaltracker.com/story/show/177600107
	// Clean up any conflicting files
	tmpDir := filepath.Dir(exe)
	for _, file := range fileutils.ListDir(tmpDir, false) {
		targetFile := filepath.Join(installPath, file)
		if fileutils.TargetExists(targetFile) {
			if err := os.Remove(targetFile); err != nil {
				return errs.Wrap(err, "Could not remove old file: %s", targetFile)
			}
		}
	}

	if err := fileutils.CopyFiles(tmpDir, installPath); err != nil {
		return errs.Wrap(err, "Failed to copy files to install dir")
	}

	shell := subshell.New(cfg)
	err = shell.WriteUserEnv(cfg, map[string]string{"PATH": installPath}, sscommon.InstallID, true)
	if err != nil {
		return errs.Wrap(err, "Could not update PATH")
	}

	if err := exeutils.ExecuteAndForget(trayInfo.Exec()); err != nil {
		return errs.Wrap(err, "Could not start %s", trayInfo.Exec())
	}

	return nil
}

func stopTrayApp(cfg *config.Instance) error {
	trayPid := cfg.GetInt(config.ConfigKeyTrayPid)
	if trayPid <= 0 {
		return nil
	}

	proc, err := process.NewProcess(int32(trayPid))
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return nil
		}
		return errs.Wrap(err, "Could not detect if state-tray pid exists")
	}
	if err := proc.Kill(); err != nil {
		return errs.Wrap(err, "Could not kill state-tray")
	}

	return nil
}