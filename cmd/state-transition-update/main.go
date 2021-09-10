package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/rollbar/rollbar-go"
)

func main() {
	var exitCode int
	defer func() {
		if panics.HandlePanics(recover(), debug.Stack()) {
			exitCode = 1
		}
		if err := events.WaitForEvents(1*time.Second, analytics.Wait, rollbar.Close, authentication.LegacyClose); err != nil {
			logging.Warning("Failed waiting to close rollbar")
		}
		os.Exit(exitCode)
	}()

	verbose := os.Getenv("VERBOSE") != ""
	logging.CurrentHandler().SetVerbose(verbose)
	logging.SetupRollbar(constants.StateToolRollbarToken)

	if err := run(); err != nil {
		logging.Error(fmt.Sprintf("%s failed with error: %s", filepath.Base(os.Args[0]), errs.Join(err, ": ")))
		fmt.Println(errs.Join(err, ": ").Error())

		exitCode = 1
		return
	}
}

func removeOldStateToolEnvironmentSettings(cfg *config.Instance) error {
	isAdmin, err := osutils.IsWindowsAdmin()
	if err != nil {
		return errs.Wrap(err, "Could not determine if running as Windows administrator")
	}

	// remove shell file additions
	s := subshell.New(cfg)
	if err := s.CleanUserEnv(cfg, sscommon.InstallID, isAdmin); err != nil {
		return errs.Wrap(err, "Failed to remove environment variable changes")

	}

	if err := s.RemoveLegacyInstallPath(cfg); err != nil {
		return errs.Wrap(err, "Failed to remove legacy install path")
	}

	return nil
}

func run() (rerr error) {
	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config")
	}
	defer rtutils.Closer(cfg.Close, &rerr)

	// The transitional State Tool will forward all commands if it can find a multi-file State Tool that has been installed in the new install location
	newInstallPath, err := pathToCheckForStateToolInstallation(cfg)
	if err != nil {
		logging.Error("Could not determine installation path to check for installation: %s", errs.JoinMessage(err))
	} else {
		newStatePath := appinfo.StateApp(newInstallPath).Exec()
		if fileutils.TargetExists(newStatePath) && newStatePath != appinfo.StateApp().Exec() {
			code, _, _ := exeutils.ExecuteAndPipeStd(newStatePath, os.Args[1:], []string{})
			os.Exit(code)
		}
	}

	switch {
	// handle state export config --filter=dir (install scripts call this function to write the install-source file)
	case len(os.Args) == 4 && os.Args[1] == "export" && os.Args[2] == "config" && os.Args[3] == "--filter=dir":
		return runExport()

	case len(os.Args) < 1 || os.Args[1] != "_prepare":
		fmt.Printf("Sorry! This is a transitional tool that should have been replaced during the last update.   If you see this message, something must have gone wrong.  Re-trying to update now. If this keeps happening please re-install the State Tool as described here: %s\n", constants.StateToolMarketingPage)
		return runDefault(cfg)

	default:
		return runDefault(cfg)
	}
}

func runExport() error {
	path, err := storage.AppDataPath()
	if err != nil {
		return errs.Wrap(err, "Failed to read app data path.")
	}
	fmt.Println(path)
	return nil
}

func runDefault(cfg *config.Instance) error {
	sessionToken := os.Getenv(constants.SessionTokenEnvVarName)
	if sessionToken != "" && cfg.GetString(analytics.CfgSessionToken) == "" {
		if err := cfg.Set(analytics.CfgSessionToken, sessionToken); err != nil {
			logging.Error("Failed to set session token: %s", errs.JoinMessage(err))
		}
		analytics.Configure(cfg)
	}

	updateTag := os.Getenv(constants.UpdateTagEnvVarName)
	if err := cfg.Set(updater.CfgUpdateTag, updateTag); err != nil {
		logging.Error("Failed to set update tag: %s", errs.JoinMessage(err))
	}

	machineid.Configure(cfg)
	machineid.SetErrorLogger(logging.Error)

	if err := removeOldStateToolEnvironmentSettings(cfg); err != nil {
		logging.Error("Failed to remove environment settings from old State Tool installation: %s", errs.JoinMessage(err))
	}

	up, err := updater.NewDefaultChecker(cfg).GetUpdateInfo("", "")
	if err != nil {
		return errs.Wrap(err, "Failed to check for latest update.")
	}

	err = up.InstallBlocking("")
	if err != nil {
		return errs.Wrap(err, "Failed to install multi-file update.")
	}

	logging.Debug("Multi-file State Tool is installed.")

	newInstallPath, err := installation.InstallPath()
	if err != nil {
		return errs.Wrap(err, "Could not get default install path")
	}

	if err := cfg.Set(installation.CfgNewInstallPath, newInstallPath); err != nil {
		return errs.Wrap(err, "Could not set new install path in config")
	}

	return nil
}

func pathToCheckForStateToolInstallation(cfg *config.Instance) (string, error) {
	newInstallPath := cfg.GetString(installation.CfgNewInstallPath)
	if newInstallPath != "" {
		return newInstallPath, nil
	}

	return installation.InstallPath()
}
