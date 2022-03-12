package svcctl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ActiveState/cli/exp/pm/internal/ipc"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/profile"
)

type SvcCtl struct {
	c *ipc.Client
}

func New(c *ipc.Client) *SvcCtl {
	return &SvcCtl{
		c: c,
	}
}

func (m *SvcCtl) Start(ctx context.Context) error {
	emsg := "start svc: %w"
	defer profile.Measure("svcmanager:Start", time.Now())

	svcExec := "../svc/build/svc" /*appinfo.SvcApp().Exec()*/
	if err := start(m.c.Namespace().AppVersion, svcExec); err != nil {
		return fmt.Errorf(emsg, err)
	}

	logging.Debug("Waiting for state-svc")
	if err := wait(m.c); err != nil {
		return fmt.Errorf(emsg, err)
	}
	return nil
}

func start(version, exec string) error {
	if !fileutils.FileExists(exec) {
		return errs.New("file %q not found", exec)
	}

	args := []string{"-v", version, "start"}

	if _, err := exeutils.ExecuteAndForget(exec, args); err != nil {
		return errs.Wrap(err, "execute and forget %q", exec)
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

	_, err := c.Ping(ctx)
	if err != nil {
		return asNotUp(err)
	}
	return nil
}
