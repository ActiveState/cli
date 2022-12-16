package rtwatcher

import (
	"errors"

	"github.com/ActiveState/cli/internal-as/analytics/dimensions"
	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/ActiveState/cli/internal-as/fileutils"
	"github.com/ActiveState/cli/internal-as/logging"
	"github.com/shirou/gopsutil/v3/process"
)

type entry struct {
	PID  int                `json:"pid"`
	Exec string             `json:"exec"`
	Dims *dimensions.Values `json:"dims"`
}

func (e entry) IsRunning() (bool, error) {
	logging.Debug("Checking if %s (%d) is still running", e.Exec, e.PID)

	proc, err := process.NewProcess(int32(e.PID))
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			logging.Debug("Process %d is no longer running", e.PID)
			return false, nil
		}
		return false, errs.Wrap(err, "Could not find process: %d", e.PID)
	}

	exe, err := proc.Exe()
	if err != nil {
		return false, errs.Wrap(err, "Could not get executable of process: %d", e.PID)
	}

	match, err := fileutils.PathsMatch(exe, e.Exec)
	if err != nil {
		return false, errs.Wrap(err, "Could not compare paths: %s, %s", exe, e.Exec)
	}
	if match {
		logging.Debug("Process %d matched", e.PID)
		return true, nil
	}

	logging.Debug("Process %d not matched, expected %s to match %s", e.PID, exe, e.Exec)
	return false, nil
}
