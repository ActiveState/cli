package svcmanager

import (
	"context"
	"time"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/pkg/platform/api/svc"
	"github.com/ActiveState/cli/pkg/platform/api/svc/request"
)

// MinimalTimeout is the minimum timeout required for requests that are meant to be near-instant
const MinimalTimeout = 500 * time.Millisecond

type Manager struct {
	ready bool
	cfg   configurable
}

type configurable interface {
	GetInt(string) int
}

func New(cfg configurable) *Manager {
	mgr := &Manager{false, cfg}
	return mgr
}

func (m *Manager) Start() error {
	defer profile.Measure("svcmanager:Start", time.Now())
	svcInfo := appinfo.SvcApp()
	if !fileutils.FileExists(svcInfo.Exec()) {
		return errs.New("Could not find: %s", svcInfo.Exec())
	}

	if _, err := exeutils.ExecuteAndForget(svcInfo.Exec(), []string{"start"}); err != nil {
		return errs.Wrap(err, "Could not start %s", svcInfo.Exec())
	}

	return nil
}

func (m *Manager) WaitWithContext(ctx context.Context) error {
	defer profile.Measure("svcmanager:WaitWithContext", time.Now())

	interrupt := false
	waitDone := make(chan struct{})
	var err error
	
	go func() {
		defer func() { waitDone <- struct{}{} }()

		logging.Debug("Waiting for state-svc")
		for try := 1; try <= 10; try++ {
			if interrupt {
				return
			}

			logging.Debug("Attempt %d", try)
			if m.Ready() {
				return
			}
			if try == 10 {
				err = locale.NewError("err_svcmanager_wait")
				return
			}

			time.Sleep(250 * time.Millisecond)
		}
	}()

	select {
	case <-waitDone:
		break
	case <-ctx.Done():
		interrupt = true
		break
	}
	return err
}

func (m *Manager) Wait() error {
	return m.WaitWithContext(context.Background())
}

func (m *Manager) Ready() bool {
	if m.ready {
		return true
	}

	if m.cfg.GetInt(constants.SvcConfigPort) == 0 {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), MinimalTimeout)
	defer cancel()
	if err := m.ping(ctx); err != nil {
		logging.Debug("Ping failed, assuming we're not ready: %v", errs.JoinMessage(err))
		return false
	}

	return true
}

func (m *Manager) ping(ctx context.Context) error {
	client, err := svc.New(m.cfg)
	if err != nil {
		return errs.Wrap(err, "Could not initialize non-retrying svc client")
	}
	r := request.NewVersionRequest()
	resp := graph.VersionResponse{}
	if err := client.RunWithContext(ctx, r, &resp); err != nil {
		return err
	}
	return nil
}
