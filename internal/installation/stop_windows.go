// +build windows

package installation

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/shirou/gopsutil/process"
	"golang.org/x/sys/windows"
)

func sendSigTerm(proc *process.Process) error {
	err := proc.SendSignal(windows.SIGTERM)
	if err != nil {
		return errs.Wrap(err, "Could not send SIGTERM signal")
	}
	return nil
}
