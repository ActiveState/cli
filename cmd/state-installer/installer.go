package main

import (
	"os"
	"path/filepath"

	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
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

func NewInstaller(cfg *config.Instance, out output.Outputer, params *Params) (*Installer, error) {
	i := &Installer{cfg: cfg, out: out, Params: params}
	if err := i.sanitize(); err != nil {
		return nil, errs.Wrap(err, "Could not sanitize input")
	}

	logging.Debug("Instantiated installer with source dir: %s, target dir: %s", i.sourcePath, i.path)

	return i, nil
}

func (i *Installer) Install() (rerr error) {
	// Store sessionToken to config
	if i.sessionToken != "" && i.cfg.GetString(anaConst.CfgSessionToken) == "" {
		if err := i.cfg.Set(anaConst.CfgSessionToken, i.sessionToken); err != nil {
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

	// Prepare bin targets is an OS specific method that will ensure we don't run into conflicts while installing
	if err := i.PrepareBinTargets(true); err != nil {
		return errs.Wrap(err, "Could not prepare for installation")
	}

	// Copy all the files
	if err := fileutils.CopyAndRenameFiles(i.sourcePath, i.path); err != nil {
		return errs.Wrap(err, "Failed to copy installation files to dir %s. Error received: %s", i.path, errs.JoinMessage(err))
	}

	// Account for v0.29 installations that use a different PATH entry
	if err := i.installDeprecationFiles(); err != nil {
		return errs.Wrap(err, "Could not install deprecation files")
	}

	// Install Launcher
	if err := i.installLauncher(); err != nil {
		return errs.Wrap(err, "Installation of system files failed.")
	}

	// Set up the environment
	binDir, err := installation.BinPathFromInstallPath(i.path)
	if err != nil {
		return errs.Wrap(err, "Could not detect installation bin path")
	}
	isAdmin, err := osutils.IsAdmin()
	if err != nil {
		return errs.Wrap(err, "Could not determine if running as Windows administrator")
	}
	shell := subshell.New(i.cfg)
	err = shell.WriteUserEnv(i.cfg, map[string]string{"PATH": binDir}, sscommon.InstallID, !isAdmin)
	if err != nil {
		return errs.Wrap(err, "Could not update PATH")
	}

	// Run state _prepare after updates to facilitate anything the new version of the state tool might need to set up
	// Yes this is awkward, followup story here: https://www.pivotaltracker.com/story/show/176507898
	if stdout, stderr, err := exeutils.ExecSimple(appinfo.StateApp(binDir).Exec(), "_prepare"); err != nil {
		logging.Error("_prepare failed after update: %v\n\nstdout: %s\n\nstderr: %s", err, stdout, stderr)
	}

	// Restart ActiveState Desktop, if it was running prior to installing
	if trayRunning {
		if _, err := exeutils.ExecuteAndForget(appinfo.TrayApp(binDir).Exec(), []string{}); err != nil {
			logging.Error("Could not start state-tray: %s", errs.JoinMessage(err))
		}
	}

	logging.Debug("Installation was successful")

	return nil
}

func PredatesBinDir() (bool, error) {
	installPath, err := installation.InstallPath()
	if err != nil {
		return false, err
	}
	binPath, err := installation.BinPath()
	if err != nil {
		return false, err
	}
	logging.Debug("PredatesBinDir: %s vs %s", installPath, binPath)
	return installPath == binPath, nil
}

func (i *Installer) installDeprecationFiles() error {
	installPath := filepath.Clean(i.InstallPath())
	binPath, err := installation.BinPathFromInstallPath(installPath)
	if err != nil {
		return errs.Wrap(err, "Could not detect whether install predates bin dir schema.")
	}
	if installPath != binPath {
		return nil
	}

	// Prepare bin targets is an OS specific method that will ensure we don't run into conflicts while installing
	if err := i.PrepareBinTargets(false); err != nil {
		return errs.Wrap(err, "Could not prepare for installation")
	}

	// Copy all the files
	if err := fileutils.CopyAndRenameFiles(filepath.Join(i.sourcePath, installation.BinDirName), i.path); err != nil {
		return errs.Wrap(err, "Failed to copy installation files to dir %s. Error received: %s", i.path, errs.JoinMessage(err))
	}

	return nil
}

func (i *Installer) InstallPath() string {
	return i.path
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
	if i.path, err = resolveInstallPath(i.path); err != nil {
		return errs.Wrap(err, "Could not resolve installation path")
	}

	// For backwards compatibility we detect the sourcePath based on the location of the installer
	if i.sourcePath == "" {
		i.sourcePath = filepath.Dir(osutils.Executable())
	}

	return nil
}
