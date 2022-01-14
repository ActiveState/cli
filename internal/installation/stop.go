package installation

import (
	"context"
	"syscall"
	"time"

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

	err = stopTray(installPath, cfg)
	if err != nil {
		return errs.Wrap(err, "Could not stop tray")
	}

	err = stopSvc(installPath)
	if err != nil {
		return errs.Wrap(err, "Could not stop service")
	}

	return nil
}

func stopTray(installPath string, cfg *config.Instance) error {
	trayInfo := appinfo.TrayApp(installPath)

	// Todo: https://www.pivotaltracker.com/story/show/177585085
	// Yes this is awkward right now
	if err := StopTrayApp(cfg); err != nil {
		return errs.Wrap(err, "Failed to stop %s", trayInfo.Name())
	}
	return nil
}

func stopSvc(installPath string) error {
	svcInfo := appinfo.SvcApp(installPath)

	if fileutils.FileExists(svcInfo.Exec()) {
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
			logging.Debug("Found running state service process: %d", p.Pid)
			err = stopSvcProcess(p)
			if err != nil {
				return errs.Wrap(err, "Could not stop service process")
			}
		}
	}

	return nil
}

func stopSvcProcess(proc *process.Process) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	done := make(chan error)
	go func() {
		done <- proc.SendSignal(syscall.SIGTERM)
	}()

	select {
	case err := <-done:
		if err != nil {
			return errs.Wrap(err, "Could not send SIGTERM to service process")
		}
		return nil
	case <-ctx.Done():
		err := proc.SendSignal(syscall.SIGKILL)
		if err != nil {
			return errs.Wrap(err, "Could not kill service process")
		}
		return nil
	}
}
