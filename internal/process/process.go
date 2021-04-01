package process

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/process"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/lockfile"
)

type Configurable interface {
	ConfigPath() string
}

// ActivationPID returns the process ID of the activated state; if any
func ActivationPID(cfg Configurable) int32 {
	pid := int32(os.Getpid())
	ppid := int32(os.Getppid())

	procInfoErrMsgFmt := "Could not detect process information: %v"

	for pid != 0 && pid != ppid {
		pidFileName := ActivationPIDFileName(cfg.ConfigPath(), int(pid))
		if fileutils.FileExists(pidFileName) {
			return pid
		}

		if ppid == 0 {
			return -1
		}

		pproc, err := process.NewProcess(ppid)
		if err != nil {
			if err != process.ErrorProcessNotRunning {
				logging.Errorf(procInfoErrMsgFmt, err)
			}
			return -1
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

// ActivationPIDFileName returns a consistent file path based on the given
// process id.
func ActivationPIDFileName(configpath string, n int) string {
	fileName := fmt.Sprintf("activation.%d", n)
	return filepath.Join(configpath, fileName)
}

// Activation eases the use of a PidLock for the purpose of "marking" a process
// as being a valid "activation".
type Activation struct {
	PIDLock *lockfile.PidLock
}

// NewActivation creates an instance of Activation.
func NewActivation(cfg Configurable, pid int) (*Activation, error) {
	pidFileName := ActivationPIDFileName(cfg.ConfigPath(), pid)
	pidLock, err := lockfile.NewPidLock(pidFileName)
	if err != nil {
		return nil, errs.Wrap(err, "cannot create new pid lock file")
	}

	locked, err := pidLock.TryLock()
	if err != nil {
		return nil, errs.Wrap(err, "cannot obtain activation pid lock")
	}

	if !locked {
		return nil, errs.New("activation pid lock is unlocked")
	}

	a := Activation{
		PIDLock: pidLock,
	}

	return &a, nil
}

// Close cleans up the used resources.
func (a *Activation) Close() error {
	return a.PIDLock.Close(false)
}

// IsActivated returns whether or not this process is being run in an activated
// state. This can be this specific process, or one of it's parents.
func IsActivated(cfg Configurable) bool {
	return ActivationPID(cfg) != -1
}
