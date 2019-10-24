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

// forwardAndExit will forward the call to the appropriate state tool version if necessary
func forwardAndExit(args []string) {
	versionInfo, fail := projectfile.ParseVersionInfo()
	if fail != nil {
		failures.Handle(fail, locale.T("err_version_parse"))
		Command.Exiter(1)
	}
	if versionInfo == nil {
		return
	}
	if !shouldForward(versionInfo) {
		return
	}

	logging.Debug("Forwarding to version %s/%s, arguments: %v", versionInfo.Branch, versionInfo.Version, args[1:])
	binary := forwardBin(versionInfo)
	ensureForwardExists(binary, versionInfo)

	execForwardAndExit(binary, args)
}

func execForwardAndExit(binary string, args []string) {
	logging.Debug("Forwarding to binary at %s", binary)

	code, _, err := osutils.ExecuteAndPipeStd(binary, args[1:], []string{fmt.Sprintf("%s=true", constants.ForwardedStateEnvVarName)})
	if err != nil {
		logging.Error("Forwarding command resulted in error: %v", err)
		print.Error(locale.Tr("forward_fail_with_error", err.Error()))
	}
	Command.Exiter(code)
}

func shouldForward(versionInfo *projectfile.VersionInfo) bool {
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

func ensureForwardExists(binary string, versionInfo *projectfile.VersionInfo) {
	if fileutils.FileExists(binary) {
		return
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
		failures.Handle(err, locale.T("forward_fail_download"))
		Command.Exiter(1)
	}

	if info == nil {
		failures.Handle(err, locale.T("forward_fail_info"))
		Command.Exiter(1)
	}

	print.Line(locale.Tr("downloading_state_version", info.Version))
	err = up.Download(binary)
	if err != nil {
		failures.Handle(err, locale.T("forward_fail_download"))
		Command.Exiter(1)
	}

	permissions, _ := permbits.Stat(binary)
	permissions.SetUserExecute(true)
	err = permbits.Chmod(binary, permissions)
	if err != nil {
		failures.Handle(err, locale.T("forward_fail_perm"))
		Command.Exiter(1)
	}
}
