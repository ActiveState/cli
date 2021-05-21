package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ActiveState/cli/cmd/state-installer/internal/installer"
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/rollbar/rollbar-go"
	"github.com/thoas/go-funk"
)

func main() {
	var exitCode int
	defer func() {
		if runbits.HandlePanics() {
			exitCode = 1
		}
		events.WaitForEvents(1*time.Second, rollbar.Close)
		os.Exit(exitCode)
	}()

	// init logging and rollbar
	verbose := os.Getenv("VERBOSE") != ""
	logging.CurrentHandler().SetVerbose(verbose)
	logging.SetupRollbar(constants.StateInstallerRollbarToken)

	out, err := output.New("plain", &output.Config{
		OutWriter:   os.Stdout,
		ErrWriter:   os.Stderr,
		Colored:     false,
		Interactive: false,
	})
	if err != nil {
		logging.Error("Could not initialize outputer: %v", err)
		exitCode = 1
		return
	}
	if err := run(out); err != nil {
		errMsg := fmt.Sprintf("%s failed with error: %s", filepath.Base(os.Args[0]), errs.Join(err, ": "))
		logging.Error(errMsg)
		out.Error(errMsg)
		out.Error(fmt.Sprintf("To retry run %s", strings.Join(os.Args, " ")))

		exitCode = 1
		return
	}
}

func run(out output.Outputer) error {
	out.Print(fmt.Sprintf("Installing version %s", constants.VersionNumber))

	cfg, err := config.Get()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config.")
	}
	machineid.SetConfiguration(cfg)
	machineid.SetErrorLogger(logging.Error)
	logging.UpdateConfig(cfg)

	var installPath string
	if len(os.Args) > 1 {
		installPath, err = filepath.Abs(os.Args[1])
		if err != nil {
			return errs.Wrap(err, "Failed to retrieve absolute installPath")
		}
	} else {
		installPath, err = installation.InstallPath()
		if err != nil {
			return errs.Wrap(err, "Failed to retrieve default installPath")
		}
	}

	if err := install(installPath, cfg, out); err != nil {
		// Todo This is running in the background, so these error messages will not be seen and only be written to the log file.
		// https://www.pivotaltracker.com/story/show/177691644
		return errs.Wrap(err, "Installing to %s failed", installPath)
	}
	logging.Debug("Installation was successful.")
	return nil
}

func install(installPath string, cfg *config.Instance, out output.Outputer) error {
	out.Print(fmt.Sprintf("Install Location: %s", installPath))
	exe, err := osutils.Executable()
	if err != nil {
		return errs.Wrap(err, "Could not detect executable path")
	}

	svcInfo := appinfo.SvcApp(installPath)
	trayInfo := appinfo.TrayApp(installPath)
	stateInfo := appinfo.StateApp(installPath)

	out.Print("Stopping services")

	// Todo: https://www.pivotaltracker.com/story/show/177585085
	// Yes this is awkward right now
	if err := installation.StopTrayApp(cfg); err != nil {
		return errs.Wrap(err, "Failed to stop %s", trayInfo.Name())
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

	tmpDir := filepath.Dir(exe)

	appDir, err := installation.LauncherInstallPath()
	if err != nil {
		return errs.Wrap(err, "Could not get system install path")
	}

	inst := installer.New(tmpDir, installPath, appDir)
	defer func() {
		os.RemoveAll(tmpDir)
		err := inst.RemoveBackupFiles()
		if err != nil {
			logging.Debug("Failed to remove backup files: %v", err)
		} else {
			logging.Debug("Removed all backup files.")
		}
	}()

	if err := inst.Install(); err != nil {
		out.Error("Installation failed, rolling back")
		rbErr := inst.Rollback()
		if rbErr != nil {
			logging.Debug("Failed to restore files: %v", rbErr)
			return errs.Wrap(err, "Installation failed and some files could not be rolled back with error: %v", rbErr)
		}
		logging.Debug("Successfully restored original files.")
		return errs.Wrap(err, "Installation failed")
	}

	out.Print("Updating environment")
	isAdmin, err := osutils.IsWindowsAdmin()
	if err != nil {
		return errs.Wrap(err, "Could not determine if running as Windows administrator")
	}
	shell := subshell.New(cfg)
	err = shell.WriteUserEnv(cfg, map[string]string{"PATH": installPath}, sscommon.InstallID, !isAdmin)
	if err != nil {
		return errs.Wrap(err, "Could not update PATH")
	}

	// Run state _prepare after updates to facilitate anything the new version of the state tool might need to set up
	// Yes this is awkward, followup story here: https://www.pivotaltracker.com/story/show/176507898
	if stdout, stderr, err := exeutils.ExecSimple(stateInfo.Exec(), "_prepare"); err != nil {
		logging.Error("_prepare failed after update: %v\n\nstdout: %s\n\nstderr: %s", err, stdout, stderr)
	}

	out.Print("Starting ActiveState Desktop")
	if _, err := exeutils.ExecuteAndForget(trayInfo.Exec(), []string{}); err != nil {
		return errs.Wrap(err, "Could not start %s", trayInfo.Exec())
	}

	out.Print("Installation Complete")

	if !funk.Contains(strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)), installPath) {
		out.Print("Please start a new shell in order to start using the ActiveState Desktop tools.")
	}

	return nil
}
