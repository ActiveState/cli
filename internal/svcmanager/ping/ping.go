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

// New returns a model that is solely used by the svcmanager to "ping" the State service to test that it is alive.
func New(ctx context.Context, cfg *config.Instance) (*Ping, error) {
	m, err := model.NewUnmanagedSvcModel(ctx, cfg)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to initialize Ping")
	}
	return &Ping{m}, nil
}

// Ping returns without an error if a client connection to the State service succeeds.
func (p *Ping) Ping() error {
	_, err := p.StateVersion()
	return err
}
