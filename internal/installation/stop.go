package installation

import (
	"context"
	"strings"
	"syscall"
	"time"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
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
		logging.Critical("Could not stop running service, error: %v", err)
		return locale.NewError("err_stop_svc", "Unable to stop state-svc process. Please manually kill any running processes with name [NOTICE]state-svc[/RESET] and try again")
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

	// This is a bit heavy handed but ensures that there are
	// no running state-svc processes which could lead to
	// errors when updating
	for _, p := range procs {
		n, err := p.Name()
		if err != nil {
			logging.Error("Could not get process name: %v", err)
			continue
		}

		if n == constants.ServiceCommandName {
			exe, err := p.Exe()
			if err != nil {
				logging.Error("Could not get executable path for state-svc process, error: %v", err)
				continue
			}

			if !strings.Contains(strings.ToLower(exe), "activestate") {
				continue
			}

			logging.Debug("Found running state-svc process: %d", p.Pid)
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

	signalErrs := make(chan error)
	go func() {
		signalErrs <- proc.SendSignal(syscall.SIGTERM)
	}()

	select {
	case err := <-signalErrs:
		if err != nil {
			return errs.Wrap(err, "Could not send SIGTERM to service process")
		}
		logging.Debug("Stopped state-svc process with SIGTERM")
		return nil
	case <-ctx.Done():
		err := proc.SendSignal(syscall.SIGKILL)
		if err != nil {
			return errs.Wrap(err, "Could not kill service process")
		}
		logging.Debug("Stopped state-svc process with SIGKILL")
		return nil
	}
}
