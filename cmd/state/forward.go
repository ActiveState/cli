package main

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/phayes/permbits"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// forceFileExt is used in tests, do not use it for anything else
var forceFileExt string

func forwardIfWarranted(args []string, out output.Outputer, pj *project.Project) (int, error) {
	if pj == nil {
		return 0, nil
	}
	if len(args) > 1 && args[1] == "update" {
		return 0, nil // Handle updates through the latest state tool version available, ie the current one
	}

	// Retrieve the version info specified in the activestate.yaml
	versionInfo, fail := projectfile.ParseVersionInfo(pj.Source().Path())
	if fail != nil {
		// if we are running `state update`, we just print the error message, but don't fail, as we can still update the state tool executable
		logging.Error("Could not parse version info from projectifle: %s", fail.Error())
		if funk.Contains(args, "update") { // Handle use case of update being called as anything but the first argument (unlikely, but possible)
			out.Error(locale.T("err_version_parse"))
			return 0, nil
		} else {
			return 1, locale.WrapError(fail, "err_version_parse", "Could not determine the State Tool version to use to run this command.")
		}
	}

	// Check if we need to forward
	if versionInfo == nil || (versionInfo.Version == constants.Version && versionInfo.Branch == constants.BranchName) {
		return 0, nil
	}

	// Perform the forward
	out.Notice(locale.Tr("forward_version", versionInfo.Version))
	code, fail := forward(args, versionInfo)
	if fail != nil {
		out.Error(locale.T("forward_fail"))
		return 1, fail
	}
	if code > 0 {
		return code, locale.NewError("err_forward", "Error occurred while running older version of the state tool, you may want to 'state update'.")
	}

	return 0, nil
}

// forward will forward the call to the appropriate State Tool version if necessary
func forward(args []string, versionInfo *projectfile.VersionInfo) (int, *failures.Failure) {
	logging.Debug("Forwarding to version %s/%s, arguments: %v", versionInfo.Branch, versionInfo.Version, args[1:])
	binary := forwardBin(versionInfo)
	fail := ensureForwardExists(binary, versionInfo)
	if fail != nil {
		return 1, fail
	}

	return execForward(binary, args)
}

func execForward(binary string, args []string) (int, *failures.Failure) {
	logging.Debug("Forwarding to binary at %s", binary)

	code, _, err := osutils.ExecuteAndPipeStd(binary, args[1:], []string{fmt.Sprintf("%s=true", constants.ForwardedStateEnvVarName)})
	if err != nil {
		logging.Error("Forwarding command resulted in error: %v", err)
		return 1, failures.FailOS.New(locale.Tr("forward_fail_with_error", err.Error()))
	}
	return code, nil
}

func forwardBin(versionInfo *projectfile.VersionInfo) string {
	filename := fmt.Sprintf("%s-%s-%s", constants.CommandName, versionInfo.Branch, versionInfo.Version)
	if forceFileExt != "" {
		filename += forceFileExt
	} else if runtime.GOOS == "windows" {
		filename += ".exe"
	}
	datadir := config.ConfigPath()
	return filepath.Join(datadir, "version-cache", filename)
}

func ensureForwardExists(binary string, versionInfo *projectfile.VersionInfo) *failures.Failure {
	if fileutils.FileExists(binary) {
		return nil
	}

	up := updater.Updater{
		CurrentVersion: constants.Version,
		APIURL:         constants.APIUpdateURL,
		CmdName:        constants.CommandName,
		DesiredBranch:  versionInfo.Branch,
		DesiredVersion: versionInfo.Version,
	}

	info, err := up.Info()
	if err != nil {
		return failures.FailNetwork.Wrap(err)
	}

	if info == nil {
		return failures.FailNetwork.New(locale.T("forward_fail_info"))
	}

	print.Line(locale.Tr("downloading_state_version", info.Version))
	err = up.Download(binary)
	if err != nil {
		return failures.FailNetwork.Wrap(err)
	}

	permissions, _ := permbits.Stat(binary)
	permissions.SetUserExecute(true)
	err = permbits.Chmod(binary, permissions)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	return nil
}
