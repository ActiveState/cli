package appinfo

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
)

type AppInfo struct {
	name       string
	executable string
	legacyExec string
}

func execDir(baseDir ...string) (resultPath string) {
	defer func() {
		// Account for legacy use-case that wasn't using the correct bin dir
		binDir := filepath.Join(resultPath, "bin")
		if fileutils.DirExists(binDir) {
			resultPath = binDir
		}
	}()

	if len(baseDir) > 0 {
		return baseDir[0]
	}

	if condition.InUnitTest() {
		// Work around tests creating a temp file, but we need the original (ie. the one from the build dir)
		rootPath := environment.GetRootPathUnsafe()
		return filepath.Join(rootPath, "build")
	}
	path, err := os.Executable()
	if err != nil {
		multilog.Error("Could not determine executable directory: %v", err)
		path, err = filepath.Abs(os.Args[0])
		if err != nil {
			multilog.Error("Could not get absolute directory of os.Args[0]", err)
		}
	}

	pathEvaled, err := filepath.EvalSymlinks(path)
	if err != nil {
		multilog.Error("Could not eval symlinks: %v", err)
	} else {
		path = pathEvaled
	}

	return filepath.Dir(path)
}

func newAppInfo(name, executableBase string, baseDir ...string) *AppInfo {
	dir := execDir(baseDir...)

	var legacyExec string
	if strings.HasSuffix(dir, "bin") {
		possibleLegacyExec := filepath.Join(filepath.Dir(dir), executableBase+osutils.ExeExt)
		if fileutils.FileExists(possibleLegacyExec) {
			legacyExec = possibleLegacyExec
		}
	}

	return &AppInfo{
		name,
		filepath.Join(dir, executableBase+osutils.ExeExt),
		legacyExec,
	}
}

func TrayApp(baseDir ...string) *AppInfo {
	return newAppInfo(constants.TrayAppName, "state-tray", baseDir...)
}

func StateApp(baseDir ...string) *AppInfo {
	return newAppInfo(constants.StateAppName, "state", baseDir...)
}

func SvcApp(baseDir ...string) *AppInfo {
	return newAppInfo(constants.SvcAppName, "state-svc", baseDir...)
}

func UpdateDialogApp(baseDir ...string) *AppInfo {
	return newAppInfo(constants.UpdateDialogName, "state-update-dialog", baseDir...)
}

func InstallerApp(baseDir ...string) *AppInfo {
	return newAppInfo(constants.StateInstallerCmd, "state-installer", baseDir...)
}

func (a *AppInfo) Name() string {
	return a.name
}

func (a *AppInfo) Exec() string {
	return a.executable
}

func (a *AppInfo) LegacyExec() string {
	return a.legacyExec
}
