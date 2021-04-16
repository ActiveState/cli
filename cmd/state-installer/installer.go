package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rollbar/rollbar-go"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
)

func main() {
	exitCode := run()
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func run() int {
	// init logging and rollbar
	verbose := os.Getenv("VERBOSE") != ""
	logging.CurrentHandler().SetVerbose(verbose)
	logging.SetupRollbar(constants.StateInstallerRollbarToken)
	defer rollbar.Close()

	cfg, err := config.New()
	if err != nil {
		logging.Error("Could not initialize config: %v", err)
		return 1
	}
	machineid.SetConfiguration(cfg)
	machineid.SetErrorLogger(logging.Error)
	logging.UpdateConfig(cfg)

	// init outputer
	out, err := output.New("plain", &output.Config{
		OutWriter:   os.Stdout,
		ErrWriter:   os.Stderr,
		Colored:     true,
		Interactive: false,
	})
	if err != nil {
		logging.Error("Failed to initialize plain outputer: %v", err)
		return 1
	}

	var installPath string
	if len(os.Args) > 1 {
		installPath = os.Args[1]
	} else {
		var err error
		installPath, err = installation.InstallPath()
		if err != nil {
			logging.Error("Failed to retrieve default installPath: %v", err)
			return 1
		}
	}

	if err := install(installPath, cfg, out); err != nil {
		// Todo This is running in the background, so these error messages will not be seen and only be written to the log file.
		// https://www.pivotaltracker.com/story/show/177691644
		errMsg := errs.Join(err, ": ").Error()
		logging.Error(errMsg)
		out.Error(errMsg)
		out.Print(fmt.Sprintf("To retry run %s", strings.Join(os.Args, " ")))
		return 1
	}
	logging.Debug("Installation was successful.")
	return 0
}

func install(installPath string, cfg *config.Instance, out output.Outputer) error {
	exe, err := osutils.Executable()
	if err != nil {
		return errs.Wrap(err, "Could not detect executable path")
	}

	svcInfo := appinfo.SvcApp(installPath)
	trayInfo := appinfo.TrayApp(installPath)
	stateInfo := appinfo.StateApp(installPath)

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
	// clean-up temp directory when we are done.
	defer os.RemoveAll(tmpDir)

	inst, err := installation.New(filepath.Join(tmpDir, "bin"), installPath)
	if err != nil {
		return errs.Wrap(err, "Could not create new installation.")
	}
	defer inst.Close()

	if err := inst.Install(); err != nil {
		restErr := inst.RestoreBackup()
		if restErr != nil {
			logging.Error("restoring of backup files failed: %v", restErr)
		}
		logging.Debug("Successfully restored original files.")
		return errs.Wrap(err, "Installation failed")
	}

	shell := subshell.New(cfg)
	err = shell.WriteUserEnv(cfg, map[string]string{"PATH": installPath}, sscommon.InstallID, true)
	if err != nil {
		return errs.Wrap(err, "Could not update PATH")
	}

	rcFile, err := shell.RcFile()
	if err == nil {
		out.Notice(fmt.Sprintf("Please either run 'source %s' or start a new login shell in order to start using the State Tool executable.", rcFile))
	} else {
		out.Notice("Please start a new login shell in order to start using the State Tool executable.")
	}

	// Run state _prepare after updates to facilitate anything the new version of the state tool might need to set up
	// Yes this is awkward, followup story here: https://www.pivotaltracker.com/story/show/176507898
	if stdout, stderr, err := exeutils.ExecSimple(stateInfo.Exec(), "_prepare"); err != nil {
		logging.Error("_prepare failed after update: %v\n\nstdout: %s\n\nstderr: %s", err, stdout, stderr)
	}

	if _, err := exeutils.ExecuteAndForget(trayInfo.Exec()); err != nil {
		return errs.Wrap(err, "Could not start %s", trayInfo.Exec())
	}

	return nil
}
