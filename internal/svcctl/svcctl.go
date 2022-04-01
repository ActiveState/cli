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
	commonTimeout = time.Millisecond * 100
)

type IPCommunicator interface {
	Requester
	Namespace() *ipc.Namespace
	PingServer(context.Context) (time.Duration, error)
	StopServer(context.Context) error
}

func NewIPCNamespaceFromGlobals() *ipc.Namespace {
	subdir := fmt.Sprintf("%s-%s", constants.CommandName, "ipc")

	return &ipc.Namespace{
		RootDir:    filepath.Join(os.TempDir(), subdir),
		AppName:    constants.CommandName,
		AppChannel: constants.BranchName,
	}
}

func NewDefaultIPCClient() *ipc.Client {
	return ipc.NewClient(NewIPCNamespaceFromGlobals())
}

func EnsureStartedAndLocateHTTP(ipComm IPCommunicator, exec string) (addr string, err error) {
	addr, err = LocateHTTP(ipComm)
	if err != nil {
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

func DefaultEnsureStartedAndLocateHTTP() (addr string, err error) {
	return EnsureStartedAndLocateHTTP(NewDefaultIPCClient(), appinfo.SvcApp().Exec())
}

func LocateHTTP(ipComm IPCommunicator) (addr string, err error) {
	comm := NewComm(ipComm)

	ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
	defer cancel()

	addr, err = comm.GetHTTPAddr(ctx)
	if err != nil {
		return "", errs.Wrap(err, "HTTP address request failed")
	}

	return addr, nil
}

func StopServer(ipComm IPCommunicator) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
	defer cancel()

	if err := stopAndWait(ctx, ipComm); err != nil {
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
	if err := waitup(ctx, ipComm); err != nil {
		return errs.Wrap(err, "Wait failed")
	}

	return nil
}

func waitup(ctx context.Context, c IPCommunicator) error {
	for try := 1; try <= 24; try++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		start := time.Now()
		timeout := time.Millisecond * time.Duration(try*try)

		logging.Debug("Attempt: %d, timeout: %v", try, timeout)
		if err := ping(ctx, c, timeout); err != nil {
			if !errors.Is(err, errNotUp) {
				return errs.Wrap(err, "Ping failed")
			}
			elapsed := time.Since(start)
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
	if err := waitdn(ctx, ipComm); err != nil {
		return errs.Wrap(err, "Wait failed")
	}

	return nil
}

func waitdn(ctx context.Context, c IPCommunicator) error {
	for try := 1; try <= 32; try++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		start := time.Now()
		timeout := time.Millisecond * time.Duration(try*try)

		logging.Debug("Attempt: %d, timeout: %v", try, timeout)
		if err := ping(ctx, c, timeout); err != nil {
			if errors.Is(err, errNotUp) {
				return nil
			}
			return errs.Wrap(err, "Ping failed")
		}
		elapsed := time.Since(start)
		time.Sleep(timeout - elapsed)
	}

	return locale.NewError("err_svcmanager_wait")
}

func ping(ctx context.Context, c IPCommunicator, timeout time.Duration) error {
	_, err := c.PingServer(ctx)
	if err != nil {
		return asNotUpError(err)
	}
	return nil
}
