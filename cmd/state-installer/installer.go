package main

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/updater"
)

type Installer struct {
	out          output.Outputer
	cfg          *config.Instance
	sessionToken string
	*Params
}

func NewInstaller(cfg *config.Instance, out output.Outputer, params *Params) *Installer {
	return &Installer{cfg: cfg, out: out, Params: params}
}

func (i *Installer) Run() error {
	if err := i.sanitize(); err != nil {
		return errs.Wrap(err, "Could not sanitize input")
	}
	return i.install()
}

func (i *Installer) install() (rerr error) {
	if err := i.PrepareBinTargets(); err != nil {
		return errs.Wrap(err, "Could not prepare for installation")
	}
 	
	// Store sessionToken to config
	if i.sessionToken != "" && i.cfg.GetString(analytics.CfgSessionToken) == "" {
		if err := i.cfg.Set(analytics.CfgSessionToken, i.sessionToken); err != nil {
			return errs.Wrap(err, "Failed to set session token")
		}
	}

	// Store update tag
	if i.updateTag != "" {
		if err := i.cfg.Set(updater.CfgUpdateTag, i.updateTag); err != nil {
			return errs.Wrap(err, "Failed to set update tag")
		}
	}

	// Stop any running processes that might interfere
	trayRunning, err := installation.IsTrayAppRunning(i.cfg)
	if err != nil {
		logging.Error("Could not determine if state-tray is running: %s", errs.JoinMessage(err))
	}
	if err := installation.StopRunning(i.path); err != nil {
		return errs.Wrap(err, "Failed to stop running services")
	}

	// Create target dir
	if err := fileutils.MkdirUnlessExists(i.path); err != nil {
		return errs.Wrap(err, "Could not create target directory: %s", i.path)
	}

	// Copy all the files
	if err := fileutils.CopyAndRenameFiles(i.sourcePath, i.path); err != nil {
		return errs.Wrap(err, "Failed to copy installation files to dir %s. Error received: %s", i.path, errs.JoinMessage(err))
	}

	// Install Launcher
	if err := i.installLauncher(); err != nil {
		return errs.Wrap(err, "Installation of system files failed.")
	}

	// Set up the environment
	isAdmin, err := osutils.IsAdmin()
	if err != nil {
		return errs.Wrap(err, "Could not determine if running as Windows administrator")
	}
	shell := subshell.New(i.cfg)
	err = shell.WriteUserEnv(i.cfg, map[string]string{"PATH": filepath.Join(i.path, "bin")}, sscommon.InstallID, !isAdmin)
	if err != nil {
		return errs.Wrap(err, "Could not update PATH")
	}

	// Run state _prepare after updates to facilitate anything the new version of the state tool might need to set up
	// Yes this is awkward, followup story here: https://www.pivotaltracker.com/story/show/176507898
	if stdout, stderr, err := exeutils.ExecSimple(appinfo.StateApp(i.path).Exec(), "_prepare"); err != nil {
		logging.Error("_prepare failed after update: %v\n\nstdout: %s\n\nstderr: %s", err, stdout, stderr)
	}

	// Restart ActiveState Desktop, if it was running prior to installing
	if trayRunning {
		if _, err := exeutils.ExecuteAndForget(appinfo.TrayApp(i.path).Exec(), []string{}); err != nil {
			logging.Error("Could not start state-tray: %s", errs.JoinMessage(err))
		}
	}

	return nil
}

// sanitize cleans up the input and inserts fallback values
func (i *Installer) sanitize() error {
	if sessionToken, ok := os.LookupEnv(constants.SessionTokenEnvVarName); ok {
		i.sessionToken = sessionToken
	}
	if tag, ok := os.LookupEnv(constants.UpdateTagEnvVarName); ok {
		i.updateTag = tag
	}

	var err error
	if i.path != "" {
		if i.path, err = filepath.Abs(i.path); err != nil {
			return errs.Wrap(err, "Failed to sanitize installation path")
		}
	} else {
		i.path, err = installation.InstallPath()
		if err != nil {
			return errs.Wrap(err, "Failed to detect default installation path")
		}
	}

	// For backwards compatibility we detect the sourcePath based on the location of the installer
	sourcePath := filepath.Dir(osutils.Executable())
	packagedStateTool := appinfo.StateApp(sourcePath)
	if i.sourcePath == "" && fileutils.FileExists(packagedStateTool.Exec()) {
		i.sourcePath = sourcePath
	}

	return nil
}
