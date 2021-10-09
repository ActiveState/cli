package report

import (
	"context"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type configurable interface {
	GetInt(string) int
}

type Report struct {
	cnf configurable
	mgr *svcmanager.Manager
}

func New(cnf configurable, mgr *svcmanager.Manager) *Report {
	return &Report{
		cnf: cnf,
		mgr: mgr,
	}
}

func (r *Report) Authentication(ctx context.Context, userID string) {
	svcmdl, err := model.NewSvcModel(context.Background(), r.cnf, r.mgr)
	if err != nil {
		logging.Errorf("Error creating service model: %v", errs.JoinMessage(err))
		return
	}

	logging.Debug("Reporting Authentication")
	if err := svcmdl.AuthenticationEvent(ctx, userID); err != nil {
		logging.Errorf("Error notifying service of updated authentication: %v", errs.JoinMessage(err))
	}
}
