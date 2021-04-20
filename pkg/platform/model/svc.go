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
	resp := graph.VersionResponse{}
	if err := m.client.Run(r, &resp); err != nil {
		return nil, err
	}
	return &resp.Version, nil
}

func (m *SvcModel) LocalProjects() ([]*graph.Project, error) {
	r := request.NewLocalProjectsRequest()
	response := graph.ProjectsResponse{[]*graph.Project{}}
	if err := m.client.Run(r, &response); err != nil {
		return nil, err
	}
	return response.Projects, nil
}

func (m *SvcModel) InitiateDeferredUpdate(channel, version string) (*graph.DeferredUpdate, error) {
	r := request.NewUpdateRequest(channel, version)
	u := graph.UpdateResponse{}
	if err := m.client.Run(r, &u); err != nil {
		return nil, errs.Wrap(err, "Error updating to version %s at channel %s", version, channel)
	}
	return &u.DeferredUpdate, nil
}
