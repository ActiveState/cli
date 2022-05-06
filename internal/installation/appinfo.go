package installation

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
)

type executableType int

const (
	StateExec executableType = iota
	ServiceExec
	TrayExec
	InstallerExec
	UpdateExec
)

var execData = map[executableType]string{
	StateExec:     constants.StateCmd + osutils.ExeExt,
	ServiceExec:   constants.StateSvcCmd + osutils.ExeExt,
	TrayExec:      constants.StateTrayCmd + osutils.ExeExt,
	InstallerExec: constants.StateInstallerCmd + osutils.ExeExt,
	UpdateExec:    constants.StateUpdateDialogCmd + osutils.ExeExt,
}

func NewExec(exec executableType) (string, error) {
	return NewExecInDir("", exec)
}

func NewExecInDir(baseDir string, exec executableType) (string, error) {
	var path string
	var err error
	if baseDir != "" {
		path, err = BinPathFromInstallPath(baseDir)
		if err != nil {
			return "", errs.Wrap(err, "Could not get bin path from base directory")
		}
	} else {
		path, err = os.Executable()
		if err != nil {
			multilog.Error("Could not determine executable: %v", err)
			path, err = filepath.Abs(os.Args[0])
			if err != nil {
				return "", errs.Wrap(err, "Could not get absolute directory of os.Args[0]")
			}
		}

		pathEvaled, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", errs.Wrap(err, "Could not eval symlinks")
		}
		path = filepath.Dir(pathEvaled)
	}

	return filepath.Join(path, execData[exec]), nil
}
