package sscommon

import (
	"os/exec"
	"strings"
	"syscall"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/osutils"
)

func stop(cmd *exec.Cmd) *failures.Failure {
	// may panic if process no longer exists
	defer failures.Recover()

	sig := syscall.SIGHUP
	if err := cmd.Process.Signal(sig); err != nil {
		return FailSignalCmd.Wrap(err)
	}

	sig = syscall.SIGTERM
	if err := cmd.Process.Signal(sig); err != nil {
		return FailSignalCmd.Wrap(err)
	}

	return nil
}

func RunFuncByBinary(binary string) RunFunc {
	bin := strings.ToLower(binary)
	if strings.Contains(bin, "bash") {
		return runWithBash
	}
	return runDirect
}

func runWithBash(env []string, name string, args ...string) (int, error) {
	filePath, fail := osutils.BashifyPath(name)
	if fail != nil {
		return 1, fail.ToError()
	}

	esc := osutils.NewBashEscaper()

	quotedArgs := filePath
	for _, arg := range args {
		quotedArgs += " " + esc.Quote(arg)
	}

	return runDirect(env, "bash", "-c", quotedArgs)
}
