package model

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/idl"
	"github.com/ActiveState/cli/pkg/platform/api/svc"
)

type SvcModel struct {
	client *svc.Client
}

func NewSvcModel(cfg *config.Instance) (*SvcModel, error) {
	client, err := svc.New(cfg)
	if err != nil {
		return nil, errs.Wrap(err, "Could not initialize svc client")
	}

	return NewSvcModelWithClient(client), nil
}

func NewSvcModelWithClient(client *svc.Client) *SvcModel {
	return &SvcModel{client}
}

func (m *SvcModel) Version() *idl.StateVersionResponse {
	return m.Version()
}
