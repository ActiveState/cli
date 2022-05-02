package appinfo

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/multilog"
)

type executable string

const (
	State     = constants.StateCmd
	Service   = constants.StateSvcCmd
	Tray      = constants.StateTrayCmd
	Installer = constants.StateInstallerCmd
	Update    = constants.StateUpdateDialogCmd
)

type appInfo struct {
	executable string
}

func NewAppInfo(exec executable) (*appInfo, error) {
	if condition.InUnitTest() {
		// Work around tests creating a temp file, but we need the original (ie. the one from the build dir)
		rootPath := environment.GetRootPathUnsafe()
		return &appInfo{executable: filepath.Join(rootPath, "build")}, nil
	}

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

	return &appInfo{
		executable: filepath.Join(filepath.Dir(path), string(exec)),
	}, nil
}

func (a *appInfo) Exec() string {
	return a.executable
}
