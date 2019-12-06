package sscommon

import (
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
		pythonPath, fail := binaryPathCMD(env, "python")
		if fail != nil {
			return 1, fail
		}
		name = pythonPath
	case ".pl":
		args = append([]string{name}, args...)
		perlPath, fail := binaryPathCMD(env, "perl")
		if fail != nil {
			return 1, fail
		}
		name = perlPath
	case ".bat":
		// No action required
	default:
		return 1, failures.FailUser.New("err_sscommon_unsupported_language", ext)
	}

	return runDirect(env, name, args...)
}

func binaryPathCMD(env []string, name string) (string, error) {
	cmd := exec.Command("where", "python")
	cmd.Env = env

	out, err := cmd.Output()
	if err != nil {
		return "", FailExecCmd.Wrap(err)
	}

	split := strings.Split(string(out), "\r\n")
	if len(split) == 0 {
		return "", failures.FailCmd.New("err_sscommon_binary_path", name)
	}

	return split[0], nil
}
