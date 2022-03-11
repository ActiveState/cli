package svcctl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ActiveState/cli/exp/pm/internal/ipc"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/profile"
)

var (
	ErrNotUp = errors.New("server not up")
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
	emsg := "svcctl: start: %w"
	defer profile.Measure("svcmanager:Start", time.Now())

	/*svcInfo := appinfo.SvcApp()
	if !fileutils.FileExists(svcInfo.Exec()) {
		return errs.New("Could not find: %s", svcInfo.Exec())
	}

	if _, err := exeutils.ExecuteAndForget(svcInfo.Exec(), []string{"start"}); err != nil {
		return errs.Wrap(err, "Could not start %s", svcInfo.Exec())
	}*/
	args := []string{"-v", m.c.Namespace().AppVersion}

	if _, err := exeutils.ExecuteAndForget("../svc/build/svc", args); err != nil {
		return fmt.Errorf(emsg, err)
	}

	logging.Debug("Waiting for state-svc")
	for try := 1; try <= 10; try++ {
		start := time.Now()
		d := time.Millisecond * 2 * time.Duration(try)

		logging.Debug("Attempt %d at %v", try, d)
		err := func() error {
			ctx, cancel := context.WithTimeout(context.Background(), d)
			defer cancel()

			_, err := m.c.Ping(ctx)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ipc.ErrServerDown) {
					return ErrNotUp
				}
				return err
			}
			return nil
		}()

		if err != nil {
			if !errors.Is(err, ErrNotUp) {
				return err
			}

			elapsed := time.Since(start)
			time.Sleep(d - elapsed)
			continue
		}
		return nil
	}

	return locale.NewError("err_svcmanager_wait")
}
