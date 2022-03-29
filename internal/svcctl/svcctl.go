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
	commonTimeout = time.Millisecond * 10
)

type IPCommunicator interface {
	Getter
	Namespace() *ipc.Namespace
	PingServer(context.Context) (time.Duration, error)
	StopServer(context.Context) error
}

func NewIPCNamespaceFromGlobals() (example *ipc.Namespace) {
	subdir := fmt.Sprintf("%s-%s", constants.CommandName, "ipc")

	return &ipc.Namespace{
		RootDir:    filepath.Join(os.TempDir(), subdir),
		AppName:    constants.CommandName,
		AppChannel: constants.BranchName,
	}
}

func EnsureAndLocateHTTP(ipComm IPCommunicator) (addr string, err error) {
	emsg := "ensure svc and locate http: %w"

	addr, err = LocateHTTP(ipComm)
	if err != nil {
		var sderr *ipc.ServerDownError
		if !errors.As(err, &sderr) {
			return "", fmt.Errorf(emsg, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
		defer cancel()

		if err := start(ctx, ipComm, appinfo.SvcApp().Exec()); err != nil {
			return "", fmt.Errorf(emsg, err)
		}

		addr, err = LocateHTTP(ipComm)
		if err != nil {
			return "", fmt.Errorf(emsg, err)
		}
	}

	return addr, nil
}

func LocateHTTP(ipComm IPCommunicator) (addr string, err error) {
	emsg := "locate http: %w"
	comm := NewComm(ipComm)

	ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
	defer cancel()

	addr, err = comm.GetHTTPAddr(ctx)
	if err != nil {
		return "", fmt.Errorf(emsg, err)
	}

	return addr, nil
}

func StopServer(ipComm IPCommunicator) (err error) {
	emsg := "stop server: %w"

	ctx, cancel := context.WithTimeout(context.Background(), commonTimeout)
	defer cancel()

	if err := stop(ctx, ipComm); err != nil {
		return fmt.Errorf(emsg, err)
	}
	return nil
}

func start(ctx context.Context, c IPCommunicator, exec string) error {
	emsg := "start svc: %w"
	defer profile.Measure("svcmanager:Start", time.Now())

	if !fileutils.FileExists(exec) {
		return errs.New("file %q not found", exec)
	}

	args := []string{"foreground"}

	if _, err := exeutils.ExecuteAndForget(exec, args); err != nil {
		return errs.Wrap(err, "execute and forget %q", exec)
	}

	logging.Debug("Waiting for state-svc")
	if err := wait(c); err != nil {
		return fmt.Errorf(emsg, err)
	}
	return nil
}

func wait(c IPCommunicator) error {
	for try := 1; try <= 16; try++ {
		start := time.Now()
		timeout := time.Millisecond * time.Duration(try*try)

		logging.Debug("Attempt: %d, timeout: %v", try, timeout)
		if err := ping(c, timeout); err != nil {
			if errors.Is(err, errNotUp) {
				elapsed := time.Since(start)
				time.Sleep(timeout - elapsed)
				continue
			}
			return err
		}
		return nil
	}

	return locale.NewError("err_svcmanager_wait")
}

func ping(c IPCommunicator, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := c.PingServer(ctx)
	if err != nil {
		return asNotUp(err)
	}
	return nil
}

func stop(ctx context.Context, c IPCommunicator) error {
	// TODO: handle errors - timeout, can't reach, etc.
	return c.StopServer(ctx)
}
