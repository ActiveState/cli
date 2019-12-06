package sscommon

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ActiveState/cli/internal/failures"
)

func stop(cmd *exec.Cmd) *failures.Failure {
	// windows should use "CTRL_CLOSE_EVENT"; SIGKILL works
	sig := syscall.SIGKILL

	// may panic if process no longer exists
	defer failures.Recover()
	if err := cmd.Process.Signal(sig); err != nil {
		return FailSignalCmd.Wrap(err)
	}

	return nil
}

func RunFuncByBinary(binary string) RunFunc {
	bin := strings.ToLower(binary)
	if strings.Contains(bin, "cmd.exe") {
		return runWithCmd
	}
	return runDirect
}

func runWithCmd(env []string, name string, args ...string) (int, error) {
	ext := filepath.Ext(name)
	switch ext {
	case ".py":
		args = append([]string{name}, args...)
		pythonPath, err := binaryPathCMD(env, "python")
		if err != nil {
			return -1, err
		}
		name = pythonPath
	case ".pl":
		args = append([]string{name}, args...)
		perlPath, err := binaryPathCMD(env, "perl")
		if err != nil {
			return -1, err
		}
		name = perlPath
	case ".bat":
		// No action required
	default:
		return -1, fmt.Errorf("unsupported language extenstion: %s", ext)
	}

	return runDirect(env, name, args...)
}
