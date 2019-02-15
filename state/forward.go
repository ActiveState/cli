package main

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"

	"github.com/phayes/permbits"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// forwardAndExit will forward the call to the appropriate state tool version if necessary
func forwardAndExit(args []string) {
	version, fail := projectfile.ParseVersion()
	if fail != nil {
		failures.Handle(fail, locale.T("err_version_parse"))
		exit(1)
	}
	if !shouldForward(version) {
		return
	}

	logging.Debug("Forwarding to version %s, arguments: %v", version, args[1:])
	binary := forwardBin(version)
	ensureForwardExists(binary, version)

	execForwardAndExit(binary, args)
}

func execForwardAndExit(binary string, args []string) {
	logging.Debug("Forwarding to binary at %s", binary)

	code, _, err := osutils.ExecuteAndPipeStd(binary, args[1:]...)
	if err != nil {
		logging.Error("Forwarding command resulted in error: %v", err)
	}
	exit(code)
}

func shouldForward(version string) bool {
	return !(version == "" || version == constants.Version)
}

func forwardBin(version string) string {
	filename := fmt.Sprintf("%s-%s", constants.CommandName, version)
	if runtime.GOOS == "windows" {
		filename += ".exe"
	}
	datadir := config.GetDataDir()
	return filepath.Join(datadir, "version-cache", filename)
}

func ensureForwardExists(binary, version string) {
	if fileutils.FileExists(binary) {
		return
	}

	up := updater.Updater{
		CurrentVersion: constants.Version,
		APIURL:         constants.APIUpdateURL,
		Dir:            constants.UpdateStorageDir,
		CmdName:        constants.CommandName,
		DesiredVersion: version,
	}

	info, err := up.Info()
	if err != nil {
		failures.Handle(err, locale.T("forward_fail_download"))
		exit(1)
	}

	if info != nil {
		print.Line(locale.Tr("downloading_state_version", version))
		err = up.Download(binary)
		if err != nil {
			failures.Handle(err, locale.T("forward_fail_download"))
			exit(1)
		}

		permissions, _ := permbits.Stat(binary)
		permissions.SetUserExecute(true)
		err = permbits.Chmod(binary, permissions)
		if err != nil {
			failures.Handle(err, locale.T("forward_fail_perm"))
			exit(1)
		}
	}
}
