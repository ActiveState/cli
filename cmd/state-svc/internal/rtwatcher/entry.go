package rtwatcher

import (
	"errors"

	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/shirou/gopsutil/process"
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

	args, err := proc.CmdlineSlice()
	if err != nil {
		return false, errs.Wrap(err, "Could not check args of process: %d", e.PID)
	}

	if len(args) == 0 {
		return false, errs.New("Process args are empty: %d", e.PID)
	}

	match, err := fileutils.PathsMatch(args[0], e.Exec)
	if err != nil {
		return false, errs.Wrap(err, "Could not compare paths: %s, %s", args[0], e.Exec)
	}
	if match {
		logging.Debug("Process %d matched", e.PID)
		return true, nil
	}

	logging.Debug("Process %d not matched, expected %s to match %s", e.PID, args[0], e.Exec)
	return false, nil
}
