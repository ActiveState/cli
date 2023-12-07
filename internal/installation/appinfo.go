package installation

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils"
)

type executableType int

const (
	state executableType = iota
	service
	installer
	executor
)

var execData = map[executableType]string{
	state:     constants.StateCmd + osutils.ExeExtension,
	service:   constants.StateSvcCmd + osutils.ExeExtension,
	installer: constants.StateInstallerCmd + osutils.ExeExtension,
	executor:  constants.StateExecutorCmd + osutils.ExeExtension,
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
		path = filepath.Dir(osutils.Executable())
	}

	// Work around dlv debugger giving an unexpected executable path
	if !condition.BuiltViaCI() && len(os.Args) > 1 && strings.Contains(os.Args[0], "__debug_bin") {
		rootPath := filepath.Clean(environment.GetRootPathUnsafe())
		if rootPath == filepath.Clean(path) {
			path = filepath.Join(path, "build")
		}
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

func InstallerExec() (string, error) {
	return newExec(installer)
}

func InstallerExecFromDir(baseDir string) (string, error) {
	return newExecFromDir(baseDir, installer)
}

func ExecutorExec() (string, error) {
	return newExec(executor)
}

func NewExecutorExecFromDir(baseDir string) (string, error) {
	return newExecFromDir(baseDir, executor)
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
