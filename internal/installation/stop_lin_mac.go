// +build !windows

package installation

import (
	"syscall"

	"github.com/shirou/gopsutil/process"
)

func sendSigTerm(proc *process.Process) error {
	return proc.SendSignal(syscall.SIGTERM)
}
