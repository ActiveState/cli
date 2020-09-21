package process

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/process"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
)

// ActivationPID returns the process ID of the activated state; if any
func ActivationPID() int32 {
	pid := int32(os.Getpid())
	ppid := int32(os.Getppid())

	procInfoErrMsgFmt := "Could not detect process information: %v"

	for ppid != 0 && pid != ppid {
		pproc, err := process.NewProcess(ppid)
		if err != nil {
			if err != process.ErrorProcessNotRunning {
				logging.Errorf(procInfoErrMsgFmt, err)
			}
			return -1
		}

		cmdArgs, err := pproc.CmdlineSlice()
		if err != nil {
			logging.Errorf(procInfoErrMsgFmt, err)
			return -1
		}

		if isActivateCmdlineArgs(cmdArgs) {
			return ppid
		}

		pid = ppid
		ppid, err = pproc.Ppid()
		if err != nil {
			logging.Errorf(procInfoErrMsgFmt, err)
			return -1
		}
	}

	return -1
}

func isActivateCmdlineArgs(args []string) bool {
	// look for the state tool command in the first argument
	exec := filepath.Base(args[0])
	if !strings.HasPrefix(exec, constants.CommandName) {
		return false
	}

	// ensure that first argument (not prefixed with a dash) is "activate"
	for _, arg := range args[1:] {
		if arg == "activate" {
			return true
		}
	}

	return false
}
