package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/gobuffalo/packr"
	"github.com/rollbar/rollbar-go"
)

func main() {
	var exitCode int
	defer func() {
		if panics.HandlePanics(recover(), debug.Stack()) {
			exitCode = 1
		}
		if err := events.WaitForEvents(1*time.Second, rollbar.Close, authentication.LegacyClose); err != nil {
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

func run() error {
	switch {
	// handle state export config --filter=dir (install scripts call this function to write the install-source file)
	case len(os.Args) == 4 && os.Args[1] == "export" && os.Args[2] == "config" && os.Args[3] == "--filter=dir":
		return runExport()

	case len(os.Args) < 1 || os.Args[1] != "_prepare":
		fmt.Printf("Sorry! This is a transitional tool that should have been replaced during the last update.   If you see this message, something must have gone wrong.  Re-trying to update now. If this keeps happening please re-install the State Tool as described here: %s\n", constants.StateToolMarketingPage)
		return runDefault()

	default:
		return runDefault()
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

func runDefault() (rerr error) {
	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config")
	}
	defer rtutils.Closer(cfg.Close, &rerr)

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
		return errs.Wrap(err, "failed to remove environment settings from old State Tool installation")
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

	if runtime.GOOS != "darwin" {
		if err := removeSelf(); err != nil {
			logging.Error("Failed to remove transitional State Tool: %s", errs.JoinMessage(err))
		}
	}

	err = addStateScript()
	if err != nil {
		logging.Error("Could not add state script: %s", errs.JoinMessage(err))
	}

	return nil
}

func addStateScript() error {
	logging.Debug("Adding state script")

	exec := appinfo.StateApp().Exec()
	script := exec
	newInstallPath, err := installation.InstallPath()
	if err != nil {
		return errs.Wrap(err, "Could not get default install path")
	}

	box := packr.NewBox("../../assets/state")
	boxFile := "state.sh"
	if runtime.GOOS == "windows" {
		boxFile = "state.bat"
		script = strings.TrimSuffix(exec, exeutils.Extension) + ".bat"
	}

	logging.Debug("NewInstallPath: %v", newInstallPath)
	tplParams := map[string]interface{}{
		"path": filepath.Join(newInstallPath, filepath.Base(exec)),
	}

	fileBytes := box.Bytes(boxFile)
	fileStr, err := strutils.ParseTemplate(string(fileBytes), tplParams)
	if err != nil {
		return errs.Wrap(err, "Could not parse %s template", boxFile)
	}

	logging.Debug("Writing to %s, value: %s", script, fileStr)

	if err = ioutil.WriteFile(script, []byte(fileStr), 0755); err != nil {
		return errs.Wrap(err, "Could not create State Tool script at %s.", script)
	}

	return nil
}
