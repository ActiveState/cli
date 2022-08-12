package installmgr

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/shirou/gopsutil/process"
)

type Configurable interface {
	GetInt(string) int
}

const ConfigKeyTrayPid = "tray-pid"

func StopTrayApp(cfg Configurable) error {
	proc, err := getTrayProcess(cfg)
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return nil
		}
		return errs.Wrap(err, "Could not detect if state-tray pid exists")
	}

	logging.Debug("Attempting to stop state-tray (%d)", proc.Pid)
	if err := proc.Kill(); err != nil {
		return errs.Wrap(err, "Could not kill state-tray")
	}

	return nil
}

func IsTrayAppRunning(cfg Configurable) (bool, error) {
	_, err := getTrayProcess(cfg)
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return false, nil
		}
		return false, errs.Wrap(err, "Could not determine if state-tray process is running")
	}

	return true, nil
}

func getTrayProcess(cfg Configurable) (*process.Process, error) {
	trayPid := cfg.GetInt(ConfigKeyTrayPid)
	if trayPid <= 0 {
		return nil, errs.Wrap(process.ErrorProcessNotRunning, "state-tray pid not set in config")
	}

	proc, err := process.NewProcess(int32(trayPid))
	if err != nil {
		return nil, errs.Wrap(err, "Could not detect if state-tray pid exists")
	}

	return proc, nil
}
