// Package svcctl provides functions that make use of an IPC device, as well as
// common IPC handlers and requesters. The intent is to guard the authority and
// uniqueness of the state service, so localized error messages refer to the
// "service" rather than just the IPC server.
package svcctl

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/profile"
)

var (
	pingRetryIterations = 45
	commonTimeout       = func() time.Duration {
		var acc int
		// alg to set max timeout matches ping backoff alg
		for i := 1; i <= pingRetryIterations; i++ {
			acc += i * i
		}
		return time.Millisecond * time.Duration(acc)
	}()
	timeBeforeNotice = 5 * time.Second
)

type IPCommunicator interface {
	Requester
	PingServer(context.Context) (time.Duration, error)
	StopServer(context.Context) error
	SockPath() *ipc.SockPath
}

type Outputer interface {
	Notice(interface{})
}

func NewIPCSockPathFromGlobals() *ipc.SockPath {
	rootDir := storage.AppDataPath()
	if os.Getenv(constants.ServiceSockDir) != "" {
		rootDir = os.Getenv(constants.ServiceSockDir)
	}

	sp := &ipc.SockPath{
		RootDir:    rootDir,
		AppName:    constants.CommandName,
		AppChannel: constants.ChannelName,
	}

	return sp
}

func NewDefaultIPCClient() *ipc.Client {
	return ipc.NewClient(NewIPCSockPathFromGlobals())
}

func EnsureExecStartedAndLocateHTTP(ipComm IPCommunicator, exec, argText string, out Outputer) (addr string, err error) {
	defer profile.Measure("svcctl:EnsureExecStartedAndLocateHTTP", time.Now())

	addr, err = LocateHTTP(ipComm)
	if err != nil {
		logging.Debug("Could not locate state-svc, attempting to start it..")

		var errServerDown *ipc.ServerDownError
		if !errors.As(err, &errServerDown) {
			return "", errs.Wrap(err, "Cannot locate HTTP port of service")
		}

		ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
		defer cancel()

		if err := startAndWait(ctx, ipComm, exec, argText, out); err != nil {
			return "", errs.Wrap(err, "Cannot start service at %q", exec)
		}

		addr, err = LocateHTTP(ipComm)
		if err != nil {
			return "", errs.Wrap(err, "Cannot locate HTTP port of service after start succeeded")
		}
	}

	return addr, nil
}

func EnsureStartedAndLocateHTTP(argText string, out Outputer) (addr string, err error) {
	svcExec, err := installation.ServiceExec()
	if err != nil {
		return "", locale.WrapError(err, "err_service_exec")
	}
	return EnsureExecStartedAndLocateHTTP(NewDefaultIPCClient(), svcExec, argText, out)
}

func LocateHTTP(ipComm IPCommunicator) (addr string, err error) {
	defer profile.Measure("svcctl:LocateHTTP", time.Now())

	comm := NewComm(ipComm)

	ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
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

	ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
	defer cancel()

	logfile, err := comm.GetLogFileName(ctx)
	if err != nil {
		return "", errs.Wrap(err, "Log file request failed")
	}

	logging.Debug("Service returned log file: %s", logfile)

	return logfile, nil
}

func StopServer(ipComm IPCommunicator) error {
	ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
	defer cancel()

	err := stopAndWait(ctx, ipComm)
	var errServerDown *ipc.ServerDownError
	if err != nil && !errors.As(err, &errServerDown) {
		return errs.Wrap(err, "Cannot stop service")
	}
	logging.Debug("service stopped")

	return nil
}

func startAndWait(ctx context.Context, ipComm IPCommunicator, exec, argText string, out Outputer) error {
	defer profile.Measure("svcmanager:Start", time.Now())

	if !fileutils.FileExists(exec) {
		return locale.NewError("svcctl_file_not_found", "", exec, constants.ForumsURL)
	}

	args := []string{"foreground", argText}
	if argText == "" {
		args = args[:len(args)-1]
	}

	if _, err := osutils.ExecuteAndForget(exec, args); err != nil {
		return locale.WrapError(err, "svcctl_cannot_exec_and_forget", "Cannot execute service in background: {{.V0}}", err.Error())
	}

	logging.Debug("Waiting for service")
	if err := waitUp(ctx, ipComm, out, newDebugData(ipComm, startSvc, argText)); err != nil {
		return locale.WrapError(err, "svcctl_wait_startup_failed", "Waiting for service startup confirmation failed")
	}

	return nil
}

var (
	waitTimeoutL10nKey = "svcctl_wait_timeout"
)

func waitUp(ctx context.Context, ipComm IPCommunicator, out Outputer, debugInfo *debugData) error {
	debugInfo.startWait()
	defer debugInfo.stopWait()

	start := time.Now()
	printedWaitingNotice := false
	for try := 1; try <= pingRetryIterations; try++ {
		select {
		case <-ctx.Done():
			err := locale.WrapError(ctx.Err(), waitTimeoutL10nKey, "", time.Since(start).String(), "1", constants.ForumsURL)
			return errs.Pack(err, debugInfo)
		default:
		}

		tryStart := time.Now()
		timeout := time.Millisecond * time.Duration(try*try)

		debugInfo.addWaitAttempt(tryStart, try, timeout)
		if err := ping(ctx, ipComm, timeout); err != nil {
			// Timeout does not reveal enough info, try again.
			// We don't need to sleep for this type of error because,
			// by definition, this is a timeout, and time has already elapsed.
			if errors.Is(err, ctlErrRequestTimeout) {
				continue
			}
			if !errors.Is(err, ctlErrNotUp) {
				err := locale.WrapError(err, "svcctl_ping_failed", "Ping encountered unexpected failure: {{.V0}}", err.Error())
				return errs.Pack(err, debugInfo)
			}
			if time.Since(start) >= timeBeforeNotice && !printedWaitingNotice && out != nil {
				out.Notice(locale.Tl("notice_waiting_state_svc", "Waiting for the State Tool service to start..."))
				printedWaitingNotice = true
			}
			elapsed := time.Since(tryStart)
			time.Sleep(timeout - elapsed)
			continue
		}
		return nil
	}

	err := locale.NewError(waitTimeoutL10nKey, "", time.Since(start).Round(time.Millisecond).String(), "2", constants.ForumsURL)
	return errs.Pack(err, debugInfo)
}

func stopAndWait(ctx context.Context, ipComm IPCommunicator) error {
	if err := ipComm.StopServer(ctx); err != nil {
		return locale.WrapError(err, "svcctl_stop_req_failed", "Service stop request failed")
	}

	logging.Debug("Waiting for service to die")
	if err := waitDown(ctx, ipComm, newDebugData(ipComm, stopSvc, "")); err != nil {
		return locale.WrapError(err, "svcctl_wait_shutdown_failed", "Waiting for service shutdown confirmation failed")
	}

	return nil
}

func waitDown(ctx context.Context, ipComm IPCommunicator, debugInfo *debugData) error {
	debugInfo.startWait()
	defer debugInfo.stopWait()

	start := time.Now()
	for try := 1; try <= pingRetryIterations; try++ {
		select {
		case <-ctx.Done():
			err := locale.WrapError(ctx.Err(), waitTimeoutL10nKey, "", time.Since(start).String(), "3", constants.ForumsURL)
			return errs.Pack(err, debugInfo)
		default:
		}

		tryStart := time.Now()
		timeout := time.Millisecond * time.Duration(try*try)

		debugInfo.addWaitAttempt(tryStart, try, timeout)
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
				err := locale.WrapError(err, "svcctl_ping_failed", "Ping encountered unexpected failure: {{.V0}}", err.Error())
				return errs.Pack(err, debugInfo)
			}
		}
		elapsed := time.Since(tryStart)
		time.Sleep(timeout - elapsed)
	}

	err := locale.NewError(waitTimeoutL10nKey, "", time.Since(start).Round(time.Millisecond).String(), "4", constants.ForumsURL)
	return errs.Pack(err, debugInfo)
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
