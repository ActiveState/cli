package appinfo

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
)

type executable int

const (
	State = iota
	Service
	Tray
	Installer
	Update
)

var appdata = map[executable]*Info{
	State:     {cmd: constants.StateCmd, name: constants.StateAppName},
	Service:   {cmd: constants.StateSvcCmd, name: constants.SvcAppName},
	Tray:      {cmd: constants.StateTrayCmd, name: constants.TrayAppName},
	Installer: {cmd: constants.StateInstallerCmd, name: constants.InstallerName},
	Update:    {cmd: constants.StateUpdateDialogCmd, name: constants.UpdateDialogName},
}

type Info struct {
	executable string
	cmd        string
	name       string
}

func New(exec executable) (*Info, error) {
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

func NewInDir(baseDir string, exec executable) (*Info, error) {
	path, err := installation.BinPathFromInstallPath(baseDir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get bin path from base directory")
	}

	info := appdata[exec]
	info.executable = filepath.Join(path, info.cmd+osutils.ExeExt)
	return info, nil
}

func (a *Info) Exec() string {
	return a.executable
}

func (a *Info) Name() string {
	return a.name
}
