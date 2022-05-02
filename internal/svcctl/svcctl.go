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
	commonTimeout = time.Millisecond * 750
)

type IPCommunicator interface {
	Requester
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
		logging.Debug("Could not locate state-svc, attempting to start it..")

		if !errs.Matches(err, &ipc.ServerDownError{}) {
			return "", errs.Wrap(err, "Cannot locate HTTP port of ipc server")
		}

		ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
		defer cancel()

		if err := startAndWait(ctx, ipComm, exec); err != nil {
			return "", errs.Wrap(err, "Cannot start ipc server at %q", exec)
		}

		addr, err = LocateHTTP(ipComm)
		if err != nil {
			return "", errs.Wrap(err, "Cannot locate HTTP port of ipc server after start succeeded")
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
		return "", errs.Wrap(err, "HTTP address request failed")
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

	logging.Debug("Log file name %s", logfile)

	return logfile, nil
}

func StopServer(ipComm IPCommunicator) error {
	ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
	defer cancel()

	err := stopAndWait(ctx, ipComm)
	if err != nil && !errs.Matches(err, &ipc.ServerDownError{}) {
		return errs.Wrap(err, "Cannot stop ipc server")
	}

	return nil
}

func startAndWait(ctx context.Context, ipComm IPCommunicator, exec string) error {
	defer profile.Measure("svcmanager:Start", time.Now())

	if !fileutils.FileExists(exec) {
		return errs.New("File %q not found", exec)
	}

	args := []string{"foreground"}

	if _, err := exeutils.ExecuteAndForget(exec, args); err != nil {
		return errs.Wrap(err, "Execute and forget %q", exec)
	}

	logging.Debug("Waiting for state-svc")
	if err := waitUp(ctx, ipComm); err != nil {
		return errs.Wrap(err, "Wait failed")
	}

	return nil
}

func waitUp(ctx context.Context, ipComm IPCommunicator) error {
	start := time.Now()
	for try := 1; try <= 32; try++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
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
				return errs.Wrap(err, "Ping failed")
			}
			elapsed := time.Since(tryStart)
			time.Sleep(timeout - elapsed)
			continue
		}
		return nil
	}

	return locale.NewError("err_svcmanager_wait")
}

func stopAndWait(ctx context.Context, ipComm IPCommunicator) error {
	if err := ipComm.StopServer(ctx); err != nil {
		return errs.Wrap(err, "IPC stop server request failed")
	}

	logging.Debug("Waiting for state-svc to die")
	if err := waitDown(ctx, ipComm); err != nil {
		return errs.Wrap(err, "Wait failed")
	}

	return nil
}

func waitDown(ctx context.Context, ipComm IPCommunicator) error {
	start := time.Now()
	for try := 1; try <= 32; try++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
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
			return errs.Wrap(err, "Ping failed")
		}
		elapsed := time.Since(tryStart)
		time.Sleep(timeout - elapsed)
	}

	return locale.NewError("err_svcmanager_wait")
}

func ping(ctx context.Context, ipComm IPCommunicator, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err := ipComm.PingServer(ctx)
	if err != nil {
		return asRequestTimeoutErr(asNotUpError(err))
	}

	return nil
}
