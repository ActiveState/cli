package ping

import (
	"context"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Ping struct {
	*model.SvcModel
}

func New(ctx context.Context, cfg *config.Instance) (*Ping, error) {
	m, err := model.NewUnmanagedSvcModel(ctx, cfg)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to initialize Ping")
	}
	return &Ping{m}, nil
}

func (p *Ping) Ping() error {
	_, err := p.StateVersion()
	return err
}
