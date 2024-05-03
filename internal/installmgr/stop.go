package installmgr

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/shirou/gopsutil/v3/process"
)

func StopRunning(installPath string) (rerr error) {
	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not get config")
	}
	defer rtutils.Closer(cfg.Close, &rerr)

	err = stopSvc(installPath)
	if err != nil {
		multilog.Critical("Could not stop running service, error: %v", errs.JoinMessage(err))
		return locale.WrapError(err, "err_stop_svc", "Unable to stop state-svc process. Please manually kill any running processes with name [NOTICE]state-svc[/RESET] and try again")
	}

	return nil
}

func stopSvc(installPath string) error {
	svcExec, err := installation.ServiceExecFromDir(installPath)
	if err != nil && !errors.Is(err, fileutils.ErrorFileNotFound) {
		return locale.WrapError(err, "err_service_exec_dir", "", installPath)
	}

	if fileutils.FileExists(svcExec) {
		exitCode, _, err := osutils.Execute(svcExec, []string{"stop"}, nil)
		if err != nil {
			// We don't return these errors because we want to fall back on killing the process
			multilog.Error("Stopping %s returned error: %s", constants.SvcAppName, errs.JoinMessage(err))
		} else if exitCode != 0 {
			multilog.Error("Stopping %s exited with code %d", constants.SvcAppName, exitCode)
		}
	}

	if condition.OnCI() { // prevent killing valid parallel instances while on CI
		return nil
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
			logging.Debug("Could not get process name: %v", err) // maybe we don't have permission
			continue
		}

		svcName := constants.ServiceCommandName + osutils.ExeExtension
		if n == svcName {
			exe, err := p.Exe()
			if err != nil {
				if runtime.GOOS == "darwin" && strings.Contains(err.Error(), "bad call to lsof") {
					// There's nothing we can do about this, so just debug log it.
					logging.Debug("Could not get executable path for state-svc process, error: %v", err)
					continue
				}
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
				if osutils.IsAccessDeniedError(err) {
					return locale.WrapExternalError(err, "err_insufficient_permissions")
				} else if errors.Is(err, os.ErrProcessDone) {
					return nil
				}
				return errs.Wrap(err, "Could not kill child process of %s", name)
			}
		}
	} else {
		logging.Error("Could not get child process: %v", err)
	}

	err = proc.Kill()
	if err != nil {
		if osutils.IsAccessDeniedError(err) {
			return locale.WrapExternalError(err, "err_insufficient_permissions")
		} else if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return errs.Wrap(err, "Could not kill %s process", name)
	}

	logging.Debug("Stopped %s process with SIGKILL", name)
	return nil
}
