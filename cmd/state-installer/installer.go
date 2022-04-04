package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
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
	if err := i.sanitizeInput(); err != nil {
		return nil, errs.Wrap(err, "Could not sanitize input")
	}

	fmt.Printf("Instantiated installer with source dir: %s, target dir: %s\n", i.sourcePath, i.path)
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
		multilog.Error("Could not determine if state-tray is running: %s", errs.JoinMessage(err))
	}
	if err := installation.StopRunning(i.path); err != nil {
		return errs.Wrap(err, "Failed to stop running services")
	}

	// Detect if existing installation needs to be cleaned
	err = detectCorruptedInstallDir(i.path)
	fmt.Println("Corrupted install:", err)
	if errors.Is(err, errCorruptedInstall) {
		err = i.sanitizeInstallPath()
		if err != nil {
			return locale.WrapError(err, "err_update_corrupt_install")
		}
	} else if err != nil {
		return locale.WrapInputError(err, "err_update_corrupt_install", constants.DocumentationURL)
	}

	// Create target dir
	if err := fileutils.MkdirUnlessExists(i.path); err != nil {
		return errs.Wrap(err, "Could not create target directory: %s", i.path)
	}

	// Prepare bin targets is an OS specific method that will ensure we don't run into conflicts while installing
	if err := i.PrepareBinTargets(); err != nil {
		return errs.Wrap(err, "Could not prepare for installation")
	}

	// Copy all the files
	if err := fileutils.CopyAndRenameFiles(i.sourcePath, i.path); err != nil {
		return errs.Wrap(err, "Failed to copy installation files to dir %s. Error received: %s", i.path, errs.JoinMessage(err))
	}

	// Install Launcher
	if err := i.installLauncher(); err != nil {
		return errs.Wrap(err, "Installation of system files failed.")
	}

	err = fileutils.Touch(filepath.Join(i.path, installation.InstallDirMarker))
	if err != nil {
		return errs.Wrap(err, "Could not place install dir marker")
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

	err = installation.SaveContext(&installation.Context{InstalledAsAdmin: isAdmin})
	if err != nil {
		return errs.Wrap(err, "Failed to set current privilege level in config")
	}

	// Run state _prepare after updates to facilitate anything the new version of the state tool might need to set up
	// Yes this is awkward, followup story here: https://www.pivotaltracker.com/story/show/176507898
	if stdout, stderr, err := exeutils.ExecSimple(appinfo.StateApp(binDir).Exec(), "_prepare"); err != nil {
		multilog.Error("_prepare failed after update: %v\n\nstdout: %s\n\nstderr: %s", err, stdout, stderr)
	}

	// Restart ActiveState Desktop, if it was running prior to installing
	if trayRunning {
		if _, err := exeutils.ExecuteAndForget(appinfo.TrayApp(binDir).Exec(), []string{}); err != nil {
			multilog.Error("Could not start state-tray: %s", errs.JoinMessage(err))
		}
	}

	logging.Debug("Installation was successful")

	return nil
}

func (i *Installer) InstallPath() string {
	return i.path
}

// sanitizeInput cleans up the input and inserts fallback values
func (i *Installer) sanitizeInput() error {
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

	return nil
}

var errCorruptedInstall = errs.New("Corrupted install")

// detectCorruptedInstallDir will return an error if it detects that the given install path is not a proper
// State Tool installation path. This mainly covers cases where we are working off of a legacy install of the State
// Tool or cases where the uninstall was not completed properly.
func detectCorruptedInstallDir(path string) error {
	if !fileutils.TargetExists(path) {
		return nil
	}

	isEmpty, err := fileutils.IsEmptyDir(path)
	if err != nil {
		return errs.Wrap(err, "Could not check if install dir is empty")
	}
	if isEmpty {
		return nil
	}

	// Detect if bin dir exists
	binPath, err := installation.BinPathFromInstallPath(path)
	if err != nil {
		return errs.Wrap(err, "Could not detect bin path")
	}
	if !fileutils.DirExists(binPath) {
		return errs.Wrap(errCorruptedInstall, "Bin path does not exist: %s", binPath)
	}

	// Detect if the install dir has files in it
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return errs.Wrap(err, "Could not read directory: %s", path)
	}

	for _, file := range files {
		if isStateExecutable(strings.ToLower(file.Name())) {
			return errs.Wrap(errCorruptedInstall, "Install directory should only contain dirs: %s", path)
		}
	}

	// Ensure that bin dir has at least the state and state-svc executables
	files, err = ioutil.ReadDir(binPath)
	if err != nil {
		return errs.Wrap(err, "Could not read bin directory: %s", path)
	}

	var found int
	for _, file := range files {
		fname := strings.ToLower(file.Name())
		if fname == constants.StateCmd+exeutils.Extension || fname == constants.StateSvcCmd+exeutils.Extension {
			found++
		}
	}

	if found != 2 {
		return errs.Wrap(errCorruptedInstall, "Bin path did not contain state tool executables.")
	}

	return nil
}

func isStateExecutable(name string) bool {
	if name == constants.StateCmd+exeutils.Extension || name == constants.StateSvcCmd+exeutils.Extension || name == constants.StateTrayCmd+exeutils.Extension {
		return true
	}
	return false
}
