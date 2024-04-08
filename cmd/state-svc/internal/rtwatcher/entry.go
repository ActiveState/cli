package rtwatcher

import (
	"errors"

	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/shirou/gopsutil/v3/process"
)

type entry struct {
	PID    int                `json:"pid"`
	Exec   string             `json:"exec"`
	Source string             `json:"source"`
	Dims   *dimensions.Values `json:"dims"`
}

// processError wraps an OS-level error, not a State Tool error.
type processError struct {
	*errs.WrapperError
}

func (e entry) IsRunning() (bool, error) {
	logging.Debug("Checking if %s (%d) is still running", e.Exec, e.PID)

	proc, err := process.NewProcess(int32(e.PID))
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			logging.Debug("Process %d is no longer running, recieved error: %s", e.PID, err.Error())
			return false, nil
		}
		return false, errs.Wrap(err, "Could not find process: %d", e.PID)
	}

	exe, err := proc.Exe()
	if err != nil {
		return false, &processError{errs.Wrap(err, "Could not get executable of process: %d", e.PID)}
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
