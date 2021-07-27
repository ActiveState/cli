package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ActiveState/cli/internal/profile"
	"github.com/phayes/permbits"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/legacyupd"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/version"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// forceFileExt is used in tests, do not use it for anything else
var forceFileExt string

const LatestVersion = "latest"

type forwardFunc func() error

func forwardFn(bindir string, args []string, out output.Outputer, pj *project.Project) (forwardFunc, error) {
	defer profile.Measure("forwarder", time.Now())
	if pj == nil {
		return nil, nil
	}
	if len(args) > 1 && args[1] == "update" {
		return nil, nil // Handle updates through the latest state tool version available, ie the current one
	}

	// Retrieve the version info specified in the activestate.yaml
	versionInfo, err := projectfile.ParseVersionInfo(pj.Source().Path())
	if err != nil {
		// if we are running `state update`, we just print the error message, but don't err, as we can still update the state tool executable
		logging.Error("Could not parse version info from projectfile: %s", err.Error())
		if funk.Contains(args, "update") { // Handle use case of update being called as anything but the first argument (unlikely, but possible)
			out.Error(locale.T("err_version_parse"))
			return nil, nil
		}

		return nil, locale.WrapError(err, "err_version_parse", "Could not determine the State Tool version to use to run this command.")
	}

	// Do we pass the version lock check?
	if versionInfo == nil || (versionInfo.Version == constants.Version && versionInfo.Branch == constants.BranchName) {
		return nil, nil
	}

	sv, err := version.ParseStateToolVersion(versionInfo.Version)
	if err != nil {
		return nil, locale.WrapInputError(err, "Failed to parse locked State Tool Version.")
	}

	// Todo Remove this block with story: https://www.pivotaltracker.com/story/show/178043272
	if !version.IsMultiFileUpdate(sv) {
		fn := func() error {
			// Perform the forward
			out.Notice(output.Heading(locale.Tl("forward_title", "Version Locked")))
			out.Notice(locale.Tr("forward_version", versionInfo.Version))
			code, err := forward(bindir, args, versionInfo, out)
			if err != nil {
				if code == 0 {
					code = 1
				}
				if errs.Matches(err, &exec.ExitError{}) {
					err = &SilencedError{err}
				}
				return locale.WrapError(err, "forward_fail")
			}
			if code > 0 {
				return errs.WrapExitCode(locale.NewError("err_forward", "Error occurred while running older version of the state tool, you may want to update the State Tool by running 'state update'."), code)
			}
			return nil
		}
		return fn, nil
	}

	updateTip := locale.Tl("lock_update_version", "Run [ACTIONABLE]state update --set-version {{.V0}}[/RESET] to change to the locked State Tool version.", versionInfo.Version)
	if !version.IsMultiFileUpdate(sv) {
		updateTip = locale.Tl("lock_update_legacy_version", "See [ACTIONABLE]{{.V0}}[/RESET] for more information on version locking and how to install a specific State Tool version.", "https://docs.activestate.com/platform/state/advanced-topics/locking/")
	}
	return nil, errs.AddTips(
		locale.NewInputError("locked_version_mismatch", "This project is locked at State Tool version {{.V0}}. Your current State Tool version is {{.V1}}.", versionInfo.Version, constants.Version),
		updateTip,
		locale.Tl("lock_update_lock", "You can lock the project to the running State Tool version with [ACTIONABLE]state update lock[/RESET]", versionInfo.Branch, versionInfo.Version),
	)
}

// forward will forward the call to the appropriate State Tool version if necessary
func forward(bindir string, args []string, versionInfo *projectfile.VersionInfo, out output.Outputer) (int, error) {
	logging.Debug("Forwarding to version %s/%s, arguments: %v", versionInfo.Branch, versionInfo.Version, args[1:])
	binary := forwardBin(bindir, versionInfo)
	err := ensureForwardExists(binary, versionInfo, out)
	if err != nil {
		return 1, err
	}

	return execForward(binary, args)
}

func execForward(binary string, args []string) (int, error) {
	logging.Debug("Forwarding to binary at %s", binary)

	code, _, err := exeutils.ExecuteAndPipeStd(binary, args[1:], []string{fmt.Sprintf("%s=true", constants.ForwardedStateEnvVarName)})
	if err != nil {
		return 1, locale.WrapError(err, "forward_fail_with_error", "", err.Error())
	}
	return code, nil
}

func forwardBin(bindir string, versionInfo *projectfile.VersionInfo) string {
	filename := fmt.Sprintf("%s-%s-%s", constants.CommandName, versionInfo.Branch, versionInfo.Version)
	if forceFileExt != "" {
		filename += forceFileExt
	} else if runtime.GOOS == "windows" {
		filename += ".exe"
	}
	return filepath.Join(bindir, "version-cache", filename)
}

func exeOverDayOld(exe string) bool {
	stat, err := os.Stat(exe)
	if err != nil {
		logging.Error("Could not stat file: %s, error: %v", exe)
		return false
	}
	diff := time.Now().Sub(stat.ModTime())
	return diff > 24*time.Hour
}

func ensureForwardExists(binary string, versionInfo *projectfile.VersionInfo, out output.Outputer) error {
	if fileutils.FileExists(binary) && (versionInfo.Version != LatestVersion || !exeOverDayOld(binary)) {
		return nil
	}

	desiredVersion := versionInfo.Version
	if desiredVersion == LatestVersion {
		desiredVersion = ""
	}

	up := legacyupd.Updater{
		CurrentVersion: constants.Version,
		APIURL:         constants.APIUpdateURL,
		DesiredBranch:  versionInfo.Branch,
		DesiredVersion: desiredVersion,
	}

	info, err := up.Info(context.Background())
	if err != nil {
		return errs.Wrap(err, "Info failed")
	}

	if info == nil {
		return locale.NewError("forward_fail_info")
	}

	out.Notice(locale.Tr("downloading_state_version", info.Version))
	err = up.Download(binary)
	if err != nil {
		return errs.Wrap(err, "Download failed")
	}

	permissions, _ := permbits.Stat(binary)
	permissions.SetUserExecute(true)
	err = permbits.Chmod(binary, permissions)
	if err != nil {
		return errs.Wrap(err, "Chmod failed")
	}

	return nil
}
