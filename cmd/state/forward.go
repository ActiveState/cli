package main

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/phayes/permbits"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// forceFileExt is used in tests, do not use it for anything else
var forceFileExt string

// forward will forward the call to the appropriate state tool version if necessary
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

func shouldForward(versionInfo *projectfile.VersionInfo) bool {
	logging.Debug("shouldForward")
	logging.Debug("Version info version: %s", versionInfo.Version)
	logging.Debug("Constants version: %s", constants.Version)
	logging.Debug("Version info branch: %s", versionInfo.Branch)
	logging.Debug("Constants branch: %s", constants.BranchName)
	return versionInfo != nil && (versionInfo.Version != constants.Version || versionInfo.Branch != constants.BranchName)
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
		Dir:            constants.UpdateStorageDir,
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
