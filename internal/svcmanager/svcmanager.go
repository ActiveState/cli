package svcmanager

import (
	"time"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/svc"
	"github.com/ActiveState/cli/pkg/platform/model"
	"golang.org/x/net/context"
)

// MinimalTimeout is the minimum timeout required for requests that are meant to be near-instant
const MinimalTimeout = 500 * time.Millisecond

type Manager struct {
	ready bool
	cfg   *config.Instance
}

func New(cfg *config.Instance) *Manager {
	mgr := &Manager{false, cfg}
	return mgr
}

func (m *Manager) Start() error {
	svcInfo := appinfo.SvcApp()
	if !fileutils.FileExists(svcInfo.Exec()) {
		return errs.New("Could not find: %s", svcInfo.Exec())
	}

	if _, err := exeutils.ExecuteAndForget(svcInfo.Exec(), []string{"start"}); err != nil {
		return errs.Wrap(err, "Could not start %s", svcInfo.Exec())
	}

	return nil
}

func (m *Manager) Wait() error {
	logging.Debug("Waiting for state-svc")
	try := 1
	for {
		logging.Debug("Attempt %d", try)
		if m.Ready() {
			return nil
		}
		if try == 10 {
			return locale.NewError("err_svcmanager_wait")
		}
		time.Sleep(time.Duration(try*100) * time.Millisecond)
		try = try + 1
	}
}

func (m *Manager) StartAndWait() error {
	if err := m.Start(); err != nil {
		return errs.Wrap(err, "Start failed during StartAndWait")
	}
	if err := m.Wait(); err != nil {
		return errs.Wrap(err, "Wait failed during StartAndWait")
	}
	return nil
}

func (m *Manager) Ready() bool {
	if m.ready {
		return true
	}

	if err := m.cfg.Reload(); err != nil {
		logging.Error("Failed to reload config: %s", errs.JoinMessage(err))
		return false
	}

	if m.cfg.GetInt(constants.SvcConfigPort) == 0 {
		return false
	}

	client, err := svc.NewWithoutRetry(m.cfg)
	if err != nil {
		logging.Error("Could not initialize svc client: %s", errs.JoinMessage(err))
		return false
	}

	model := model.NewSvcModelWithClient(context.Background(), client)
	_, err = model.StateVersion()
	if err != nil {
		logging.Debug("StateVersion failed, assuming we're not ready: %v", errs.JoinMessage(err))
		return false
	}

	return true
}

func (m *Manager) Model(ctx context.Context) (*model.SvcModel, error) {
	if err := m.Wait(); err != nil {
		return nil, errs.Wrap(err, "Wait failed before initializing svc model.")
	}

	svc, err := model.NewSvcModel(ctx, m.cfg)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to initialize svc model.")

	}
	return svc, nil
}
