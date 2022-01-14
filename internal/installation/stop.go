package installation

import (
	"fmt"
	"syscall"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/shirou/gopsutil/process"
)

func StopRunning(installPath string) (rerr error) {
	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not get config")
	}
	defer rtutils.Closer(cfg.Close, &rerr)

	svcInfo := appinfo.SvcApp(installPath)
	trayInfo := appinfo.TrayApp(installPath)

	// Todo: https://www.pivotaltracker.com/story/show/177585085
	// Yes this is awkward right now
	if err := StopTrayApp(cfg); err != nil {
		return errs.Wrap(err, "Failed to stop %s", trayInfo.Name())
	}

	fmt.Println("Looking for exec at:", svcInfo.Exec())
	if fileutils.FileExists(svcInfo.Exec()) {
		fmt.Println(fmt.Sprintf("running command: %s %s", svcInfo.Exec(), "stop"))
		exitCode, _, err := exeutils.Execute(svcInfo.Exec(), []string{"stop"}, nil)
		if err != nil {
			return errs.Wrap(err, "Stopping %s returned error", svcInfo.Name())
		}
		if exitCode != 0 {
			return errs.New("Stopping %s exited with code %d", svcInfo.Name(), exitCode)
		}
	}

	procs, err := process.Processes()
	if err != nil {
		return errs.Wrap(err, "Could not get list of running processes")
	}

	for _, p := range procs {
		n, err := p.Name()
		if err != nil {
			logging.Error("Could not get process name: %v", err)
			continue
		}

		if n == constants.ServiceCommandName {
			fmt.Println("Found running proc")
			err = p.SendSignal(syscall.SIGINT)
			if err != nil {
				return errs.Wrap(err, "Could not stop state-svc process")
			}
		}
	}

	return nil
}
