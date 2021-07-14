package installation

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/shirou/gopsutil/process"
)

type Configurable interface {
	GetInt(string) int
}

const ConfigKeyTrayPid = "tray-pid"

func StopTrayApp(cfg Configurable) error {
	trayPid := cfg.GetInt(ConfigKeyTrayPid)
	if trayPid <= 0 {
		return nil
	}

	proc, err := process.NewProcess(int32(trayPid))
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return nil
		}
		return errs.Wrap(err, "Could not detect if state-tray pid exists")
	}
	if err := proc.Kill(); err != nil {
		return errs.Wrap(err, "Could not kill state-tray")
	}

	return nil
}
