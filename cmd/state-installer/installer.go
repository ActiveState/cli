package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ActiveState/cli/cmd/state-installer/internal/installer"
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/rollbar/rollbar-go"
	"github.com/thoas/go-funk"
)

func main() {
	var exitCode int
	an := analytics.New()
	defer func() {
		if panics.HandlePanics(recover(), debug.Stack()) {
			exitCode = 1
		}
		if err := events.WaitForEvents(1*time.Second, an.Wait, rollbar.Close, authentication.LegacyClose); err != nil {
			logging.Warning("Failed to wait for rollbar to close: %v", err)
		}
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
		logging.Critical("Could not initialize outputer: %v", err)
		exitCode = 1
		return
	}
	installPath := ""
	if len(os.Args) > 1 {
		installPath = os.Args[1]
	}
	var updateTag *string
	tag, ok := os.LookupEnv(constants.UpdateTagEnvVarName)
	if ok {
		updateTag = &tag
	}
	if err := run(out, installPath, os.Getenv(constants.SessionTokenEnvVarName), updateTag); err != nil {
		errMsg := fmt.Sprintf("%s failed with error: %s", filepath.Base(os.Args[0]), errs.Join(err, ": "))
		logging.Critical(errMsg)
		out.Error(errMsg)
		out.Error(fmt.Sprintf("To retry run %s", strings.Join(os.Args, " ")))

		exitCode = 1
		return
	}
}

func run(out output.Outputer, installPath, sessionToken string, updateTag *string) (rerr error) {
	out.Print(fmt.Sprintf("Installing version %s", constants.VersionNumber))

	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config.")
	}
	defer rtutils.Closer(cfg.Close, &rerr)

	machineid.Configure(cfg)
	machineid.SetErrorLogger(logging.Error)

	if sessionToken != "" && cfg.GetString(analytics.CfgSessionToken) == "" {
		if err := cfg.Set(analytics.CfgSessionToken, sessionToken); err != nil {
			logging.Error("Failed to set session token: %s", errs.JoinMessage(err))
		}
	}

	if updateTag != nil {
		if err := cfg.Set(updater.CfgUpdateTag, *updateTag); err != nil {
			logging.Error("Failed to set update tag: %s", errs.JoinMessage(err))
		}
	}

	if installPath != "" {
		installPath, err = filepath.Abs(installPath)
		if err != nil {
			return errs.Wrap(err, "Failed to retrieve absolute installPath")
		}
	} else {
		installPath, err = installation.InstallPath()
		if err != nil {
			return errs.Wrap(err, "Failed to retrieve default installPath")
		}
	}

	logging.Debug("Installing to %s", installPath)
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

	trayInfo := appinfo.TrayApp(installPath)
	stateInfo := appinfo.StateApp(installPath)

	trayRunning, err := installation.IsTrayAppRunning(cfg)
	if err != nil {
		logging.Error("Could not determine if state-tray is running: %v", err)
	}

	out.Print("Stopping services")

	if err := installation.StopRunning(installPath); err != nil {
		return errs.Wrap(err, "Failed to stop running services")
	}

	tmpDir := filepath.Dir(exe)

	appDir, err := installation.LauncherInstallPath()
	if err != nil {
		return errs.Wrap(err, "Could not get system install path")
	}

	inst := installer.New(tmpDir, installPath, appDir)
	defer os.RemoveAll(tmpDir)

	if err := inst.Install(); err != nil {
		out.Error("Installation failed.")
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

	if trayRunning {
		out.Print("Starting ActiveState Desktop")
		if _, err := exeutils.ExecuteAndForget(trayInfo.Exec(), []string{}); err != nil {
			return errs.Wrap(err, "Could not start %s", trayInfo.Exec())
		}
	}

	out.Print("Installation Complete")

	_, isForward := os.LookupEnv(constants.ForwardedStateEnvVarName)
	if !isForward && !funk.Contains(strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)), installPath) {
		out.Print("Please start a new shell in order to start using the State Tool.")
	}

	return nil
}
