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
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/pkg/platform/api/svc"
	"github.com/ActiveState/cli/pkg/platform/api/svc/request"
)

// MinimalTimeout is the minimum timeout required for requests that are meant to be near-instant
const MinimalTimeout = 500 * time.Millisecond

type errVersionMismatch struct{ *locale.LocalizedError }

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

	logging.Debug("Waiting for state-svc")
	for try := 1; try <= 10; try++ {
		logging.Debug("Attempt %d", try)
		select {
		case <-ctx.Done():
			return nil
		default:
			err := m.Ready()
			if err == nil {
				return nil
			}
			if errs.Matches(err, errVersionMismatch{}) {
				return errs.Wrap(err, "Incorrect State Service version")
			}
			logging.Debug("Ready failed, assuming we're not ready: %v", errs.JoinMessage(err))
		}
		time.Sleep(250 * time.Millisecond)
	}

	return locale.NewError("err_svcmanager_wait")
}

func (m *Manager) Wait() error {
	return m.WaitWithContext(context.Background())
}

func (m *Manager) Ready() error {
	if m.ready {
		return nil
	}

	if m.cfg.GetInt(constants.SvcConfigPort) == 0 {
		return errs.New("Service port not set in config")
	}

	ctx, cancel := context.WithTimeout(context.Background(), MinimalTimeout)
	defer cancel()
	if err := m.ping(ctx); err != nil {
		return errs.Wrap(err, "Ping failed, service may not yet be ready")
	}

	return nil
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

	if !rtutils.BuiltViaCI {
		return nil
	}

	if resp.Version.State.Version != constants.Version && resp.Version.State.Branch != constants.BranchName {
		return errVersionMismatch{locale.NewError("err_ping_version_mismatch")}
	}

	return nil
}
