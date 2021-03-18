package model

import (
	"context"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/pkg/platform/api/svc"
	"github.com/ActiveState/cli/pkg/platform/api/svc/request"
)

type SvcModel struct {
	ctx    context.Context
	client *gqlclient.Client
}

func NewSvcModel(ctx context.Context, cfg *config.Instance) (*SvcModel, error) {
	client, err := svc.New(cfg)
	if err != nil {
		return nil, errs.Wrap(err, "Could not initialize svc client")
	}
	return &SvcModel{ctx, client}, nil
}

func (m *SvcModel) StateVersion() (*graph.Version, error) {
	r := request.NewVersionRequest()
	v := &graph.Version{}
	if err := m.client.Run(r, &v); err != nil {
		return nil, err
	}
	return v, nil
}
