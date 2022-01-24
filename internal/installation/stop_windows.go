// +build windows

package installation

import (
	"github.com/shirou/gopsutil/process"
	"golang.org/x/sys/windows"
)

func sendSigTerm(proc *process.Process) error {
	return proc.SendSignal(windows.SIGTERM)
}
