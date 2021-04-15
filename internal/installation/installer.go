package installation

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/shirou/gopsutil/process"
)

func InstallPath() (string, error) {
	// Facilitate use-case of running executables from the build dir while developing
	if !rtutils.BuiltViaCI && strings.Contains(path.Clean(os.Args[0]), "/build/") {
		return filepath.Dir(os.Args[0]), nil
	}
	return defaultInstallPath()
}

func StopTrayApp(cfg *config.Instance) error {
	trayPid := cfg.GetInt(config.ConfigKeyTrayPid)
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

func LogfilePath(configPath string, pid int) string {
	return filepath.Join(configPath, fmt.Sprintf("state-installer-%d.log", pid))
}
