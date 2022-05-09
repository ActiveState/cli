package installation

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils"
)

type executableType int

const (
	state executableType = iota
	service
	tray
	installer
	update
)

var execData = map[executableType]string{
	state:     constants.StateCmd + osutils.ExeExt,
	service:   constants.StateSvcCmd + osutils.ExeExt,
	tray:      constants.StateTrayCmd + osutils.ExeExt,
	installer: constants.StateInstallerCmd + osutils.ExeExt,
	update:    constants.StateUpdateDialogCmd + osutils.ExeExt,
}

func newExec(exec executableType) (string, error) {
	return newExecFromDir("", exec)
}

func newExecFromDir(baseDir string, exec executableType) (string, error) {
	var path string
	var err error
	if baseDir != "" {
		path, err = BinPathFromInstallPath(baseDir)
		if err != nil {
			return "", errs.Wrap(err, "Could not get bin path from base directory")
		}
	} else {
		path = osutils.Executable()
	}

	return filepath.Join(path, execData[exec]), nil
}

func StateExec() (string, error) {
	return newExec(state)
}

func StateExecFromDir(baseDir string) (string, error) {
	return newExecFromDir(baseDir, state)
}

func ServiceExec() (string, error) {
	return newExec(service)
}

func ServiceExecFromDir(baseDir string) (string, error) {
	return newExecFromDir(baseDir, service)
}

func TrayExec() (string, error) {
	return newExec(tray)
}

func TrayExecFromDir(baseDir string) (string, error) {
	return newExecFromDir(baseDir, tray)
}

func InstallerExec() (string, error) {
	return newExec(installer)
}

func InstallerExecFromDir(baseDir string) (string, error) {
	return newExecFromDir(baseDir, installer)
}

func UpdateExec() (string, error) {
	return newExec(update)
}

func NewUpdateExecFromDir(baseDir string) (string, error) {
	return newExecFromDir(baseDir, update)
}

func Executables() ([]string, error) {
	var execs []string
	for _, data := range execData {
		exec, err := newExec(installer)
		if err != nil {
			return nil, errs.Wrap(err, "Could not get executable data for command: %s", data)
		}
		execs = append(execs, exec)
	}
	return execs, nil
}
