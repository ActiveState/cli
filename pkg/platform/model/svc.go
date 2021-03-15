package model

import (
	"context"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/idl"
	"github.com/ActiveState/cli/pkg/platform/api/svc"
)

type SvcModel struct {
	c   idl.VersionSvcClient // should remain private
	ctx context.Context
}

type VersionResponse struct {
	*idl.StateVersionResponse
}

func NewSvcModel(ctx context.Context, cfg *config.Instance) (*SvcModel, error) {
	client, err := svc.New(cfg)
	if err != nil {
		return nil, errs.Wrap(err, "Could not initialize svc client")
	}
	return NewSvcModelWithClient(ctx, client), nil
}

func NewSvcModelWithClient(ctx context.Context, client idl.VersionSvcClient) *SvcModel {
	return &SvcModel{
		c:   client,
		ctx: ctx,
	}
}

func (m *SvcModel) StateVersion() (*VersionResponse, error) {
	res, err := m.c.StateVersion(m.ctx, &idl.StateVersionRequest{})
	if err != nil {
		return nil, errs.Wrap(err, "Request failed")
	}
	return &VersionResponse{res}, nil
}
