package model

import (
	"context"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/svc"
	"github.com/ActiveState/cli/pkg/platform/api/svc/request"
)

type SvcModel struct {
	ctx    context.Context
	client *gqlclient.Client
}

type ConnectionWaiter interface {
	Wait() error
}

func NewSvcModel(ctx context.Context, cfg *config.Instance, svcWait ConnectionWaiter) (*SvcModel, error) {
	client, err := svc.New(cfg)
	if err != nil {
		return nil, errs.Wrap(err, "Could not initialize svc client")
	}

	if err := svcWait.Wait(); err != nil {
		return nil, errs.Wrap(err, "Failed to wait for svc connection to be ready")
	}

	return NewSvcModelWithClient(ctx, client), nil
}

func NewSvcModelWithClient(ctx context.Context, client *gqlclient.Client) *SvcModel {
	return &SvcModel{ctx, client}
}

func (m *SvcModel) StateVersion() (*graph.Version, error) {
	r := request.NewVersionRequest()
	resp := graph.VersionResponse{}
	if err := m.client.RunWithContext(m.ctx, r, &resp); err != nil {
		return nil, err
	}
	return &resp.Version, nil
}

func (m *SvcModel) LocalProjects() ([]*graph.Project, error) {
	r := request.NewLocalProjectsRequest()
	response := graph.ProjectsResponse{Projects: []*graph.Project{}}
	if err := m.client.RunWithContext(m.ctx, r, &response); err != nil {
		return nil, err
	}
	return response.Projects, nil
}

func (m *SvcModel) InitiateDeferredUpdate(channel, version string) (*graph.DeferredUpdate, error) {
	r := request.NewUpdateRequest(channel, version)
	u := graph.UpdateResponse{}
	if err := m.client.RunWithContext(m.ctx, r, &u); err != nil {
		return nil, locale.WrapError(err, "err_svc_updaterequest", "Error updating to version {{.V0}} at channel {{.V1}}: {{.V2}}", version, channel, errs.Join(err, ": ").Error())
	}
	return &u.DeferredUpdate, nil
}

func (m *SvcModel) CheckUpdate() (*graph.AvailableUpdate, error) {
	r := request.NewAvailableUpdate()
	u := graph.AvailableUpdateResponse{}
	if err := m.client.RunWithContext(m.ctx, r, &u); err != nil {
		return nil, errs.Wrap(err, "Error checking if update is available.")
	}

	// Todo: https://www.pivotaltracker.com/story/show/178205825
	if u.AvailableUpdate.Version == "" {
		return nil, nil
	}
	return &u.AvailableUpdate, nil
}
