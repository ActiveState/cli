package svcctl

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/svccomm"
)

var (
	svcExec = func() string {
		var fileExt string
		if runtime.GOOS == "windows" {
			fileExt = ".exe"
		}
		return filepath.Clean("../svc/build/svc" + fileExt) /*appinfo.SvcApp().Exec()*/
	}()
)

func EnsureAndLocateHTTP(n *ipc.Namespace) (addr string, err error) {
	ipcClient := ipc.NewClient(n)
	emsg := "ensure svc and locate http: %w"
	commClient := svccomm.NewClient(ipcClient)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*2)
	defer cancel()

	addr, err = commClient.GetHTTPAddr(ctx)
	if err != nil {
		var sderr *ipc.ServerDownError
		if !errors.As(err, &sderr) {
			return "", fmt.Errorf(emsg, err)
		}

		fmt.Println("starting service")
		ctx1, cancel1 := context.WithTimeout(context.Background(), time.Millisecond*2)
		defer cancel1()

		if err := start(ctx1, ipcClient, svcExec); err != nil {
			return "", fmt.Errorf(emsg, err)
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), time.Millisecond*2)
		defer cancel2()
		addr, err = commClient.GetHTTPAddr(ctx2)
		if err != nil {
			return "", fmt.Errorf(emsg, err)
		}
	}

	return addr, nil
}

func LocateHTTP(n *ipc.Namespace) (addr string, err error) {
	ipcClient := ipc.NewClient(n)
	emsg := "locate http: %w"
	commClient := svccomm.NewClient(ipcClient)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*2)
	defer cancel()

	addr, err = commClient.GetHTTPAddr(ctx)
	fmt.Println(addr)
	if err != nil {
		return "", fmt.Errorf(emsg, err)
	}

	return addr, nil
}

func StopServer(n *ipc.Namespace) (err error) {
	ipcClient := ipc.NewClient(n)
	emsg := "stop server: %w"

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*2)
	defer cancel()

	if err := stop(ctx, ipcClient); err != nil {
		return fmt.Errorf(emsg, err)
	}
	return nil
}

func start(ctx context.Context, c *ipc.Client, exec string) error {
	emsg := "start svc: %w"
	defer profile.Measure("svcmanager:Start", time.Now())

	if !fileutils.FileExists(exec) {
		return errs.New("file %q not found", exec)
	}

	args := []string{"-v", c.Namespace().AppVersion, "start"}

	if _, err := exeutils.ExecuteAndForget(exec, args); err != nil {
		return errs.Wrap(err, "execute and forget %q", exec)
	}

	logging.Debug("Waiting for state-svc")
	if err := wait(c); err != nil {
		return fmt.Errorf(emsg, err)
	}
	return nil
}

func wait(c *ipc.Client) error {
	for try := 1; try <= 16; try++ {
		start := time.Now()
		d := time.Millisecond * time.Duration(try*try)

		logging.Debug("Attempt %d at %v", try, d)
		if err := ping(c, d); err != nil {
			if errors.Is(err, errNotUp) {
				elapsed := time.Since(start)
				time.Sleep(d - elapsed)
				continue
			}
			return err
		}
		return nil
	}

	return locale.NewError("err_svcmanager_wait")
}

func ping(c *ipc.Client, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	_, err := c.PingServer(ctx)
	if err != nil {
		return asNotUp(err)
	}
	return nil
}

func stop(ctx context.Context, c *ipc.Client) error {
	// TODO: handle errors - timeout, can't reach, etc.
	return c.StopServer(ctx)
}
