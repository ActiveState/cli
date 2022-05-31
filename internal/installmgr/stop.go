package installmgr

import (
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
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
		multilog.Critical("Could not stop running service, error: %v", errs.JoinMessage(err))
		return locale.WrapError(err, "err_stop_svc", "Unable to stop state-svc process. Please manually kill any running processes with name [NOTICE]state-svc[/RESET] and try again")
	}

	return nil
}

func stopTray(installPath string, cfg *config.Instance) error {
	// Todo: https://www.pivotaltracker.com/story/show/177585085
	// Yes this is awkward right now
	if err := StopTrayApp(cfg); err != nil {
		return errs.Wrap(err, "Failed to stop %s", constants.TrayAppName)
	}
	return nil
}

func stopSvc(installPath string) error {
	svcExec, err := installation.ServiceExecFromDir(installPath)
	if err != nil {
		return locale.WrapError(err, "err_service_exec_dir", "", installPath)
	}

	if fileutils.FileExists(svcExec) {
		exitCode, _, err := exeutils.Execute(svcExec, []string{"stop"}, nil)
		if err != nil {
			// We don't return these errors because we want to fall back on killing the process
			multilog.Error("Stopping %s returned error: %s", constants.SvcAppName, errs.JoinMessage(err))
		} else if exitCode != 0 {
			multilog.Error("Stopping %s exited with code %d", constants.SvcAppName, exitCode)
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
			multilog.Error("Could not get process name: %v", err)
			continue
		}

		svcName := constants.ServiceCommandName + exeutils.Extension
		if n == svcName {
			exe, err := p.Exe()
			if err != nil {
				multilog.Error("Could not get executable path for state-svc process, error: %v", err)
				continue
			}

			if !strings.Contains(strings.ToLower(exe), "activestate") {
				multilog.Error("Found state-svc process in unexpected directory: %s", exe)
				continue
			}

			logging.Debug("Found running state-svc process with PID %d, at %s", p.Pid, exe)
			err = stopSvcProcess(p, n)
			if err != nil {
				return errs.Wrap(err, "Could not stop service process")
			}
		}
	}

	return nil
}

func stopSvcProcess(proc *process.Process, name string) error {
	// Process library does not have support for sending signals to Windows processes
	if runtime.GOOS == "windows" {
		return killProcess(proc, name)
	}

	signalErrs := make(chan error)
	go func() {
		signalErrs <- proc.SendSignal(syscall.SIGTERM)
	}()

	select {
	case err := <-signalErrs:
		if err != nil {
			multilog.Error("Could not send SIGTERM to %s  process, error: %v", name, err)
			return killProcess(proc, name)
		}

		running, err := proc.IsRunning()
		if err != nil {
			return errs.Wrap(err, "Could not check if %s is still running, error: %v", name, err)
		}
		if running {
			return killProcess(proc, name)
		}

		logging.Debug("Stopped %s process with SIGTERM", name)
		return nil
	case <-time.After(time.Second):
		return killProcess(proc, name)
	}
}

func killProcess(proc *process.Process, name string) error {
	children, err := proc.Children()
	if err == nil {
		for _, c := range children {
			err = c.Kill()
			if err != nil {
				return errs.Wrap(err, "Could not kill child process of %s", name)
			}
		}
	} else {
		logging.Error("Could not get child process: %v", err)
	}

	err = proc.Kill()
	if err != nil {
		return errs.Wrap(err, "Could not kill %s process", name)
	}

	logging.Debug("Stopped %s process with SIGKILL", name)
	return nil
}
