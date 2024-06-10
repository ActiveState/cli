//go:build !darwin
// +build !darwin

package camel

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rollbar"
)

// InstallFromArchive will unpack the installer archive, locate the install script, and then use the installer
// script to install an ActivePython runtime to the configured runtime dir. Any failures
// during this process will result in a failed installation and the install-dir being removed.
func (m *metaData) pythonRelocationDir(installRoot string) (string, error) {
	python, err := locatePythonExecutable(installRoot)
	if err != nil {
		return "", err
	}

	prefix, err := extractPythonRelocationPrefix(installRoot, python)
	if err != nil {
		return "", err
	}

	// relocate python
	return prefix, nil
}

// locatePythonExecutable will locate the path to the python binary in the runtime dir.
func locatePythonExecutable(installDir string) (string, error) {
	binPath := filepath.Join(installDir, "bin")
	python2 := filepath.Join(installDir, "bin", constants.ActivePython2Executable)
	python3 := filepath.Join(installDir, "bin", constants.ActivePython3Executable)

	var executablePath string
	if fileutils.FileExists(python3) {
		executablePath = python3
	} else if fileutils.FileExists(python2) {
		executablePath = python2
	} else {
		return "", locale.NewError("installer_err_runtime_no_executable", "", binPath, constants.ActivePython2Executable, constants.ActivePython3Executable)
	}

	if !fileutils.IsExecutable(executablePath) {
		return "", errs.New("Executable '%s' does not have execute permissions", executablePath)
	}
	return executablePath, nil
}

// extractRelocationPrefix will extract the prefix that needs to be replaced for this installation.
func extractPythonRelocationPrefix(installDir string, python string) (string, error) {
	prefixBytes, err := exec.Command(python, "-c", "import activestate; print('\\n'.join(activestate.prefixes))").Output()
	logging.Debug("bin: %s", python)
	logging.Debug("OUTPUT: %s", string(prefixBytes))
	if err != nil {
		if exitErr, isExitError := err.(*exec.ExitError); isExitError {
			multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("obtaining relocation prefixes: %v : %s", err, string(prefixBytes))
			exitError := exitErr
			return "", errs.Wrap(err, "python import prefixes failed with exit error: %s", exitError.String())
		}
		return "", errs.Wrap(err, "python import prefixes failed")
	}
	if strings.TrimSpace(string(prefixBytes)) == "" {
		return "", errs.Wrap(err, "Received empty prefix")
	}
	return strings.Split(string(prefixBytes), "\n")[0], nil
}
