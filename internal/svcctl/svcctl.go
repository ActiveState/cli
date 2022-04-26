// Package svcctl provides functions that make use of an IPC device, as well as
// common IPC handlers and requesters.
package svcctl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/profile"
)

var (
	pingRetryIterations = 32
	commonTimeout       = func() time.Duration {
		var acc int
		// alg to set max timeout matches ping backoff alg
		for i := 1; i <= pingRetryIterations; i++ {
			acc += i * i
		}
		return time.Millisecond * time.Duration(acc)
	}()
)

type IPCommunicator interface {
	Requester
	SockPath() *ipc.SockPath
	PingServer(context.Context) (time.Duration, error)
	StopServer(context.Context) error
}

func NewIPCSockPathFromGlobals() *ipc.SockPath {
	subdir := fmt.Sprintf("%s-%s", constants.CommandName, "ipc")

	return &ipc.SockPath{
		RootDir:    filepath.Join(os.TempDir(), subdir),
		AppName:    constants.CommandName,
		AppChannel: constants.BranchName,
	}
}

func NewDefaultIPCClient() *ipc.Client {
	return ipc.NewClient(NewIPCSockPathFromGlobals())
}

func EnsureExecStartedAndLocateHTTP(ipComm IPCommunicator, exec string) (addr string, err error) {
	addr, err = LocateHTTP(ipComm)
	if err != nil {
		if !errs.Matches(err, &ipc.ServerDownError{}) {
			return "", err
		}

		ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
		defer cancel()

		if err := startAndWait(ctx, ipComm, exec); err != nil {
			return "", err
		}

		addr, err = LocateHTTP(ipComm)
		if err != nil {
			return "", err
		}
	}

	return addr, nil
}

func EnsureStartedAndLocateHTTP() (addr string, err error) {
	return EnsureExecStartedAndLocateHTTP(NewDefaultIPCClient(), appinfo.SvcApp().Exec())
}

func LocateHTTP(ipComm IPCommunicator) (addr string, err error) {
	comm := NewComm(ipComm)

	ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
	defer cancel()

	addr, err = comm.GetHTTPAddr(ctx)
	if err != nil {
		return "", locale.WrapError(err, "svcctl_http_addr_req_fail", "Request for service HTTP address failed")
	}

	return addr, nil
}

func StopServer(ipComm IPCommunicator) error {
	ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
	defer cancel()

	err := stopAndWait(ctx, ipComm)
	if err != nil && !errs.Matches(err, &ipc.ServerDownError{}) {
		return err
	}

	return nil
}

func startAndWait(ctx context.Context, ipComm IPCommunicator, exec string) error {
	defer profile.Measure("svcmanager:Start", time.Now())

	if !fileutils.FileExists(exec) {
		return locale.NewError("svcctl_file_not_found", "Service communications file not found")
	}

	args := []string{"foreground"}

	if _, err := exeutils.ExecuteAndForget(exec, args); err != nil {
		return locale.WrapError(err, "svcctl_cannot_exec_and_forget", "Cannot execute service in background: {{.V0}}", err.Error())
	}

	logging.Debug("Waiting for service")
	if err := waitUp(ctx, ipComm); err != nil {
		return locale.WrapError(err, "svcctl_wait_startup_failed", "Waiting for service startup confirmation failed")
	}

	return nil
}

var (
	waitTimeoutL10nKey = "svcctl_wait_timeout"
	waitTimeoutL10nVal = "Timed out waiting for service to start ({{.V0}}). Are you running software that could prevent State Tool from running local processes/servers?"
)

func waitUp(ctx context.Context, ipComm IPCommunicator) error {
	start := time.Now()
	for try := 1; try <= pingRetryIterations; try++ {
		select {
		case <-ctx.Done():
			return locale.WrapError(ctx.Err(), waitTimeoutL10nKey, waitTimeoutL10nVal, time.Since(start).String())
		default:
		}

		tryStart := time.Now()
		timeout := time.Millisecond * time.Duration(try*try)

		logging.Debug("Attempt: %d, timeout: %v, total: %v", try, timeout, time.Since(start))
		if err := ping(ctx, ipComm, timeout); err != nil {
			// Timeout does not reveal enough info, try again.
			// We don't need to sleep for this type of error because,
			// by definition, this is a timeout, and time has already elapsed.
			if errors.Is(err, ctlErrRequestTimeout) {
				continue
			}
			if !errors.Is(err, ctlErrNotUp) {
				return locale.WrapError(err, "svcctl_ping_failed", "Ping encountered unexpected failure: {{.V0}}", err.Error())
			}
			elapsed := time.Since(tryStart)
			time.Sleep(timeout - elapsed)
			continue
		}
		return nil
	}

	return locale.NewError(waitTimeoutL10nKey, waitTimeoutL10nVal, time.Since(start).Round(time.Millisecond).String())
}

func stopAndWait(ctx context.Context, ipComm IPCommunicator) error {
	if err := ipComm.StopServer(ctx); err != nil {
		return locale.WrapError(err, "svcctl_stop_req_failed", "Service stop request failed")
	}

	logging.Debug("Waiting for service to die")
	if err := waitDown(ctx, ipComm); err != nil {
		return locale.WrapError(err, "svcctl_wait_shutdown_failed", "Waiting for service shutdown confirmation failed")
	}

	return nil
}

func waitDown(ctx context.Context, ipComm IPCommunicator) error {
	start := time.Now()
	for try := 1; try <= pingRetryIterations; try++ {
		select {
		case <-ctx.Done():
			return locale.WrapError(ctx.Err(), waitTimeoutL10nKey, waitTimeoutL10nVal, time.Since(start).String())
		default:
		}

		tryStart := time.Now()
		timeout := time.Millisecond * time.Duration(try*try)

		logging.Debug("Attempt: %d, timeout: %v, total: %v", try, timeout, time.Since(start))
		if err := ping(ctx, ipComm, timeout); err != nil {
			// Timeout does not reveal enough info, try again.
			// We don't need to sleep for this type of error because,
			// by definition, this is a timeout, and time has already elapsed.
			if errors.Is(err, ctlErrRequestTimeout) {
				continue
			}
			if errors.Is(err, ctlErrNotUp) {
				return nil
			}
			return locale.WrapError(err, "svcctl_ping_failed", "Ping encountered unexpected failure: {{.V0}}", err.Error())
		}
		elapsed := time.Since(tryStart)
		time.Sleep(timeout - elapsed)
	}

	return locale.NewError(waitTimeoutL10nKey, waitTimeoutL10nVal, time.Since(start).Round(time.Millisecond).String())
}

func ping(ctx context.Context, ipComm IPCommunicator, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err := ipComm.PingServer(ctx)
	if err != nil {
		return asRequestTimeoutCtlErr(asNotUpCtlErr(err))
	}

	return nil
}
