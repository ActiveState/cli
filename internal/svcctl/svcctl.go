// Package svcctl provides functions that make use of an IPC device, as well as
// common IPC handlers and requesters. The intent is to guard the authority and
// uniqueness of the state service, so localized error messages refer to the
// "service" rather than just the IPC server.
package svcctl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/profile"
)

var (
	pingRetryIterations = 26
	baseStart           = 8
	modifierNumerator   = 9
	modifierDenominator = 10

	ipCommTimeout     = time.Second * 4
	upDownWaitTimeout = func() time.Duration {
		var acc int
		// alg to set backstop timeout (matches ping backoff alg)
		// last iter does not count towards total, so -1 the iters
		for base := baseStart; base < pingRetryIterations+baseStart-1; base++ {
			acc += ((base * base) * modifierNumerator) / modifierDenominator
		}
		return time.Millisecond*time.Duration(acc) + time.Second // add buffer
	}()
)

type IPCommunicator interface {
	Requester
	PingServer(context.Context) (time.Duration, error)
	StopServer(context.Context) error
	SockPath() *ipc.SockPath
}

func NewIPCSockPathFromGlobals() *ipc.SockPath {
	subdir := fmt.Sprintf("%s-%s", constants.CommandName, "ipc")
	rootDir := filepath.Join(os.TempDir(), subdir)
	if os.Getenv(constants.ServiceSockDir) != "" {
		rootDir = os.Getenv(constants.ServiceSockDir)
	}

	return &ipc.SockPath{
		RootDir:    rootDir,
		AppName:    constants.CommandName,
		AppChannel: constants.BranchName,
	}
}

func NewDefaultIPCClient() *ipc.Client {
	return ipc.NewClient(NewIPCSockPathFromGlobals())
}

func EnsureExecStartedAndLocateHTTP(ipComm IPCommunicator, exec, argText string) (addr string, err error) {
	defer profile.Measure("svcctl:EnsureExecStartedAndLocateHTTP", time.Now())

	addr, err = LocateHTTP(ipComm)
	if err != nil {
		logging.Debug("Could not locate state-svc, attempting to start it..")

		if !errs.Matches(err, &ipc.ServerDownError{}) {
			return "", errs.Wrap(err, "Cannot locate HTTP port of service")
		}

		if err := startAndWait(ipComm, exec, argText); err != nil {
			return "", errs.Wrap(err, "Cannot start service at %q", exec)
		}

		addr, err = LocateHTTP(ipComm)
		if err != nil {
			return "", errs.Wrap(err, "Cannot locate HTTP port of service after start succeeded")
		}
	}

	return addr, nil
}

func EnsureStartedAndLocateHTTP(argText string) (addr string, err error) {
	svcExec, err := installation.ServiceExec()
	if err != nil {
		return "", locale.WrapError(err, "err_service_exec")
	}
	return EnsureExecStartedAndLocateHTTP(NewDefaultIPCClient(), svcExec, argText)
}

func LocateHTTP(ipComm IPCommunicator) (addr string, err error) {
	defer profile.Measure("svcctl:LocateHTTP", time.Now())

	comm := NewComm(ipComm)

	ctx, cancel := context.WithTimeout(context.Background(), ipCommTimeout)
	defer cancel()

	addr, err = comm.GetHTTPAddr(ctx)
	if err != nil {
		return "", locale.WrapError(err, "svcctl_http_addr_req_fail", "Request for service HTTP address failed")
	}

	logging.Debug("Located state-svc at %s", addr)

	return addr, nil
}

func LogFileName(ipComm IPCommunicator) (string, error) {
	comm := NewComm(ipComm)

	ctx, cancel := context.WithTimeout(context.Background(), ipCommTimeout)
	defer cancel()

	logfile, err := comm.GetLogFileName(ctx)
	if err != nil {
		return "", errs.Wrap(err, "Log file request failed")
	}

	logging.Debug("Service returned log file: %s", logfile)

	return logfile, nil
}

func StopServer(ipComm IPCommunicator) error {
	err := stopAndWait(ipComm)
	if err != nil && !errs.Matches(err, &ipc.ServerDownError{}) {
		return errs.Wrap(err, "Cannot stop service")
	}

	return nil
}

func startAndWait(ipComm IPCommunicator, exec, argText string) error {
	defer profile.Measure("svcmanager:Start", time.Now())

	if !fileutils.FileExists(exec) {
		return locale.NewError("svcctl_file_not_found", constants.ForumsURL)
	}

	args := []string{"foreground", argText}
	if argText == "" {
		args = args[:len(args)-1]
	}

	wdd := newWaitDebugData(ipComm.SockPath(), execSvc)
	if _, err := exeutils.ExecuteAndForget(exec, args); err != nil {
		return locale.WrapError(
			err, "svcctl_cannot_exec_and_forget",
			"Cannot execute service in background: {{.V0}}", err.Error(),
		)
	}
	wdd.stampExec()

	logging.Debug("ExecuteAndForget took %v", wdd.execDur)

	ctx, cancel := context.WithTimeout(context.Background(), upDownWaitTimeout)
	defer cancel()

	logging.Debug("Waiting for service")
	wdd.startWait()
	err := waitUp(ctx, ipComm, wdd)
	wdd.stampWait()
	logging.Debug("Wait duration: %s", wdd.waitDur)
	if err != nil {
		return locale.WrapError(
			err, "svcctl_wait_startup_failed",
			"Waiting for service startup confirmation failed after {{.V0}}",
			time.Since(wdd.waitStart).String(),
		)
	}

	return nil
}

var (
	waitTimeoutL10nKey = "svcctl_wait_timeout"
)

func waitUp(ctx context.Context, ipComm IPCommunicator, wdd *waitDebugData) error {
	for try := 1; try <= pingRetryIterations; try++ {
		base := try - 1 + baseStart
		select {
		case <-ctx.Done():
			err := locale.WrapError(ctx.Err(), waitTimeoutL10nKey, "", "1", constants.ForumsURL)
			return errs.Pack(err, wdd)
		default:
		}

		tryStart := time.Now()
		timeout := time.Millisecond * time.Duration(
			((base*base)*modifierNumerator)/modifierDenominator,
		)

		wa := newWaitAttempt(wdd.waitStart, tryStart, try, timeout)
		wdd.addAttempts(wa)

		logging.Debug("%s", wa)

		if err := ping(ctx, ipComm, timeout); err != nil {
			// Timeout does not reveal enough info, try again.
			// We don't need to sleep for this type of error because,
			// by definition, this is a timeout, and time has already elapsed.
			if errors.Is(err, ctlErrRequestTimeout) {
				continue
			}
			if !errors.Is(err, ctlErrNotUp) {
				err := locale.WrapError(
					err, "svcctl_ping_failed",
					"Ping encountered unexpected failure: {{.V0}}",
					err.Error(),
				)
				return errs.Pack(err, wdd)
			}

			if try < pingRetryIterations {
				elapsed := time.Since(tryStart)
				time.Sleep(timeout - elapsed)
			}

			continue
		}

		return nil
	}

	err := locale.NewError(waitTimeoutL10nKey, "", "2", constants.ForumsURL)
	return errs.Pack(err, wdd)
}

func stopAndWait(ipComm IPCommunicator) error {
	stopCtx, stopCancel := context.WithTimeout(context.Background(), ipCommTimeout)
	defer stopCancel()

	wdd := newWaitDebugData(ipComm.SockPath(), stopSvc)
	if err := ipComm.StopServer(stopCtx); err != nil {
		return locale.WrapError(err, "svcctl_stop_req_failed", "Service stop request failed")
	}
	wdd.stampExec()

	waitCtx, waitCancel := context.WithTimeout(context.Background(), upDownWaitTimeout)
	defer waitCancel()

	logging.Debug("Waiting for service to die")
	wdd.startWait()
	err := waitDown(waitCtx, ipComm, wdd)
	wdd.stampWait()
	logging.Debug("Wait duration: %s", wdd.waitDur)
	if err != nil {
		return locale.WrapError(
			err, "svcctl_wait_shutdown_failed",
			"Waiting for service shutdown confirmation failed after {{.V0}}",
			time.Since(wdd.waitStart).String(),
		)
	}

	return nil
}

func waitDown(ctx context.Context, ipComm IPCommunicator, wdd *waitDebugData) error {
	for try := 1; try <= pingRetryIterations; try++ {
		base := try - 1 + baseStart
		select {
		case <-ctx.Done():
			err := locale.WrapError(ctx.Err(), waitTimeoutL10nKey, "", "3", constants.ForumsURL)
			return errs.Pack(err, wdd)
		default:
		}

		tryStart := time.Now()
		timeout := time.Millisecond * time.Duration(
			((base*base)*modifierNumerator)/modifierDenominator,
		)

		wa := newWaitAttempt(wdd.waitStart, tryStart, try, timeout)
		wdd.addAttempts(wa)

		logging.Debug("%s", wa)

		if err := ping(ctx, ipComm, timeout); err != nil {
			// Timeout does not reveal enough info, try again.
			// We don't need to sleep for this type of error because,
			// by definition, this is a timeout, and time has already elapsed.
			if errors.Is(err, ctlErrRequestTimeout) {
				continue
			}
			if errors.Is(err, io.EOF) {
				continue
			}
			if errors.Is(err, ctlErrNotUp) {
				return nil
			}
			if !errors.Is(err, ctlErrTempNotUp) {
				err := locale.WrapError(
					err, "svcctl_ping_failed",
					"Ping encountered unexpected failure: {{.V0}}",
					err.Error(),
				)
				return errs.Pack(err, wdd)
			}
		}

		if try < pingRetryIterations {
			elapsed := time.Since(tryStart)
			time.Sleep(timeout - elapsed)
		}
	}

	err := locale.NewError(waitTimeoutL10nKey, "", "4", constants.ForumsURL)
	return errs.Pack(err, wdd)
}

func ping(ctx context.Context, ipComm IPCommunicator, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err := ipComm.PingServer(ctx)
	if err != nil {
		return asRequestTimeoutCtlErr(asNotUpCtlErr(asTempNotUpCtlErr(err)))
	}

	return nil
}
