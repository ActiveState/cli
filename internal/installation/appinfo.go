package installation

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
)

type executable int

const (
	StateApp = iota
	ServiceApp
	TrayApp
	InstallerApp
	UpdateApp
)

var appdata = map[executable]*AppInfo{
	StateApp:     {cmd: constants.StateCmd, name: constants.StateAppName},
	ServiceApp:   {cmd: constants.StateSvcCmd, name: constants.SvcAppName},
	TrayApp:      {cmd: constants.StateTrayCmd, name: constants.TrayAppName},
	InstallerApp: {cmd: constants.StateInstallerCmd, name: constants.InstallerName},
	UpdateApp:    {cmd: constants.StateUpdateDialogCmd, name: constants.UpdateDialogName},
}

type AppInfo struct {
	executable string
	cmd        string
	name       string
}

func NewAppInfo(exec executable) (*AppInfo, error) {
	path, err := os.Executable()
	if err != nil {
		multilog.Error("Could not determine executable: %v", err)
		path, err = filepath.Abs(os.Args[0])
		if err != nil {
			return nil, errs.Wrap(err, "Could not get absolute directory of os.Args[0]")
		}
	}

	pathEvaled, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, errs.Wrap(err, "Could not eval symlinks")
	}
	path = pathEvaled

	info := appdata[exec]
	info.executable = filepath.Join(filepath.Dir(path), info.cmd+osutils.ExeExt)
	return info, nil
}

func NewAppInfoInDir(baseDir string, exec executable) (*AppInfo, error) {
	var path string
	var err error
	if condition.InUnitTest() {
		// Work around tests creating a temp file, but we need the original (ie. the one from the build dir)
		rootPath := environment.GetRootPathUnsafe()
		path = filepath.Join(rootPath, "build")
	} else {
		path, err = BinPathFromInstallPath(baseDir)
		if err != nil {
			return nil, errs.Wrap(err, "Could not get bin path from base directory")
		}
	}

	info := appdata[exec]
	info.executable = filepath.Join(path, info.cmd+osutils.ExeExt)

	return info, nil
}

func (a *AppInfo) Exec() string {
	return a.executable
}

func (a *AppInfo) Name() string {
	return a.name
}
