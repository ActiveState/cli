// +build !windows

package installation

import (
	"syscall"

	"github.com/shirou/gopsutil/process"
)

func sendSigTerm(proc *process.Process) error {
	err := proc.SendSignal(syscall.SIGTERM)
	if err != nil {
		return errs.Wrap(err, "Could not send SIGTERM signal")
	}
	return nil
}
