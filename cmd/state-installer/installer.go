package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	svcApp "github.com/ActiveState/cli/cmd/state-svc/app"
	svcAutostart "github.com/ActiveState/cli/cmd/state-svc/autostart"
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installmgr"
	"github.com/ActiveState/cli/internal/legacytray"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/updater"
)

type Installer struct {
	out         output.Outputer
	cfg         *config.Instance
	an          analytics.Dispatcher
	payloadPath string
	*Params
}

func NewInstaller(cfg *config.Instance, out output.Outputer, an analytics.Dispatcher, payloadPath string, params *Params) (*Installer, error) {
	i := &Installer{cfg: cfg, out: out, an: an, payloadPath: payloadPath, Params: params}
	if err := i.sanitizeInput(); err != nil {
		return nil, errs.Wrap(err, "Could not sanitize input")
	}

	logging.Debug("Instantiated installer with source dir: %s, target dir: %s", i.payloadPath, i.path)

	return i, nil
}

func (i *Installer) Install() (rerr error) {
	isAdmin, err := osutils.IsAdmin()
	if err != nil {
		return errs.Wrap(err, "Could not determine if running as Windows administrator")
	}
	if isAdmin && !i.Params.isUpdate {
		prompter := prompt.New(i.out, i.an)
		if i.Params.nonInteractive {
			prompter.SetInteractive(false)
		}
		if i.Params.force {
			prompter.SetForce(true)
		}
		defaultChoice := i.Params.nonInteractive
		confirm, err := prompter.Confirm("", locale.T("installer_prompt_is_admin"), &defaultChoice, ptr.To(true))
		if err != nil {
			return errs.Wrap(err, "Not confirmed")
		}
		if !confirm {
			return locale.NewInputError("installer_aborted", "Installation aborted by the user")
		}
	}

	// Store update tag
	if i.updateTag != "" {
		if err := i.cfg.Set(updater.CfgUpdateTag, i.updateTag); err != nil {
			return errs.Wrap(err, "Failed to set update tag")
		}
	}

	// Stop any running processes that might interfere
	if err := installmgr.StopRunning(i.path); err != nil {
		return errs.Wrap(err, "Failed to stop running services")
	}

	// Detect if existing installation needs to be cleaned
	err = detectCorruptedInstallDir(i.path)
	if errors.Is(err, errCorruptedInstall) {
		err = i.sanitizeInstallPath()
		if err != nil {
			return locale.WrapError(err, "err_update_corrupt_install")
		}
	} else if err != nil {
		return locale.WrapInputError(err, "err_update_corrupt_install", constants.DocumentationURL)
	}

	err = legacytray.DetectAndRemove(i.path, i.cfg)
	if err != nil {
		multilog.Error("Unable to detect and/or remove legacy tray. Will try again next update. Error: %v", err)
	}

	// Create target dir
	if err := fileutils.MkdirUnlessExists(i.path); err != nil {
		return errs.Wrap(err, "Could not create target directory: %s", i.path)
	}

	// Prepare bin targets is an OS specific method that will ensure we don't run into conflicts while installing
	if err := i.PrepareBinTargets(); err != nil {
		return errs.Wrap(err, "Could not prepare for installation")
	}

	// Copy all the files except for the current executable
	if err := fileutils.CopyAndRenameFiles(i.payloadPath, i.path, filepath.Base(osutils.Executable())); err != nil {
		if osutils.IsAccessDeniedError(err) {
			// If we got to this point, we could not copy and rename over existing files.
			// This is a permission issue. (We have an installer test for copying and renaming over a file
			// in use, which does not raise an error.)
			return locale.WrapExternalError(err, "err_update_access_denied", "", errs.JoinMessage(err))
		}
		return errs.Wrap(err, "Failed to copy installation files to dir %s. Error received: %s", i.path, errs.JoinMessage(err))
	}

	// Set up the environment
	binDir := filepath.Join(i.path, installation.BinDirName)

	// Install the state service as an app if necessary
	if err := i.installSvcApp(binDir); err != nil {
		return errs.Wrap(err, "Installation of service app failed.")
	}

	// Configure available shells
	shell := subshell.New(i.cfg)
	err = subshell.ConfigureAvailableShells(shell, i.cfg, map[string]string{"PATH": binDir}, sscommon.InstallID, !isAdmin)
	if err != nil {
		return errs.Wrap(err, "Could not configure available shells")
	}

	err = installation.SaveContext(&installation.Context{InstalledAsAdmin: isAdmin})
	if err != nil {
		return errs.Wrap(err, "Failed to set current privilege level in config")
	}

	stateExec, err := installation.StateExecFromDir(binDir)
	if err != nil {
		return locale.WrapError(err, "err_state_exec")
	}

	// Run state _prepare after updates to facilitate anything the new version of the state tool might need to set up
	// Yes this is awkward, followup story here: https://www.pivotaltracker.com/story/show/176507898
	if stdout, stderr, err := osutils.ExecSimple(stateExec, []string{"_prepare"}, []string{}); err != nil {
		multilog.Error("_prepare failed after update: %v\n\nstdout: %s\n\nstderr: %s", err, stdout, stderr)
	}

	logging.Debug("Installation was successful")

	return nil
}

func (i *Installer) InstallPath() string {
	return i.path
}

// sanitizeInput cleans up the input and inserts fallback values
func (i *Installer) sanitizeInput() error {
	if tag, ok := os.LookupEnv(constants.UpdateTagEnvVarName); ok {
		i.updateTag = tag
	}

	var err error
	if i.path, err = resolveInstallPath(i.path); err != nil {
		return errs.Wrap(err, "Could not resolve installation path")
	}

	return nil
}

func (i *Installer) installSvcApp(binDir string) error {
	app, err := svcApp.NewFromDir(binDir)
	if err != nil {
		return errs.Wrap(err, "Could not create app")
	}

	err = app.Install()
	if err != nil {
		return errs.Wrap(err, "Could not install app")
	}

	if err = autostart.Upgrade(app.Path(), svcAutostart.Options); err != nil {
		return errs.Wrap(err, "Failed to upgrade autostart for service app.")
	}

	if err = autostart.Enable(app.Path(), svcAutostart.Options); err != nil {
		return errs.Wrap(err, "Failed to enable autostart for service app.")
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

	// Detect if the install dir has files in it
	files, err := os.ReadDir(path)
	if err != nil {
		return errs.Wrap(err, "Could not read directory: %s", path)
	}

	// Executable files should be in bin dir, not root dir
	for _, file := range files {
		if isStateExecutable(strings.ToLower(file.Name())) {
			return errs.Wrap(errCorruptedInstall, "Install directory should only contain dirs: %s", path)
		}
	}

	return nil
}

func isStateExecutable(name string) bool {
	if name == constants.StateCmd+osutils.ExeExtension || name == constants.StateSvcCmd+osutils.ExeExtension {
		return true
	}
	return false
}

func installedOnPath(installRoot, channel string) (bool, string, error) {
	if !fileutils.DirExists(installRoot) {
		return false, "", nil
	}

	// This is not using appinfo on purpose because we want to deal with legacy installation formats, which appinfo does not
	stateCmd := constants.StateCmd + osutils.ExeExtension

	// Check for state.exe in channel, root and bin dir
	// This is to handle older state tool versions that gave incompatible input paths
	// Also, fall back on checking for the install dir marker in case of a failed uninstall attempt.
	candidates := []string{
		filepath.Join(installRoot, channel, installation.BinDirName, stateCmd),
		filepath.Join(installRoot, channel, stateCmd),
		filepath.Join(installRoot, installation.BinDirName, stateCmd),
		filepath.Join(installRoot, stateCmd),
		filepath.Join(installRoot, installation.InstallDirMarker),
	}
	for _, candidate := range candidates {
		if fileutils.TargetExists(candidate) {
			return true, installRoot, nil
		}
	}

	return false, installRoot, nil
}
