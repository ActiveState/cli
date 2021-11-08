package model

import (
	"context"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/platform/api/svc"
	"github.com/ActiveState/cli/pkg/platform/api/svc/request"
)

type SvcModel struct {
	svcm   *svcmanager.Manager
	cfg    *config.Instance
	client *svc.Client
}

// NewSvcModel returns a model for all client connections to a State Svc.  This function returns an error if the State service is not yet ready to communicate.
func NewSvcModel(cfg *config.Instance, svcm *svcmanager.Manager) *SvcModel {
	return &SvcModel{cfg: cfg, svcm: svcm}
}

func (m *SvcModel) request(ctx context.Context, request gqlclient.Request, resp interface{}) error {
	defer profile.Measure("SvcModel:request", time.Now())
	if m.client == nil {
		if err := m.svcm.WaitWithContext(ctx); err != nil {
			return errs.Wrap(err, "Failed to wait for svc connection to be ready")
		}

		client, err := svc.New(m.cfg)
		if err != nil {
			return errs.Wrap(err, "Could not initialize svc client")
		}
		m.client = client
	}

	return m.client.RunWithContext(ctx, request, resp)
}

func (m *SvcModel) StateVersion(ctx context.Context) (*graph.Version, error) {
	r := request.NewVersionRequest()
	resp := graph.VersionResponse{}
	if err := m.request(ctx, r, &resp); err != nil {
		return nil, err
	}
	return &resp.Version, nil
}

func (m *SvcModel) LocalProjects(ctx context.Context) ([]*graph.Project, error) {
	r := request.NewLocalProjectsRequest()
	response := graph.ProjectsResponse{Projects: []*graph.Project{}}
	if err := m.request(ctx, r, &response); err != nil {
		return nil, err
	}
	return response.Projects, nil
}

func (m *SvcModel) CheckUpdate(ctx context.Context) (*graph.AvailableUpdate, error) {
	defer profile.Measure("svc:CheckUpdate", time.Now())
	r := request.NewAvailableUpdate()
	u := graph.AvailableUpdateResponse{}
	if err := m.request(ctx, r, &u); err != nil {
		return nil, errs.Wrap(err, "Error checking if update is available.")
	}

	// Todo: https://www.pivotaltracker.com/story/show/178205825
	if u.AvailableUpdate.Version == "" {
		return nil, nil
	}
	return &u.AvailableUpdate, nil
}

func (m *SvcModel) Ping() error {
	_, err := m.StateVersion(context.Background())
	return err
}

func (m *SvcModel) AnalyticsEvent(ctx context.Context, category, action, label string, dimJson string) error {
	defer profile.Measure("svc:analyticsEvent", time.Now())

	r := request.NewAnalyticsEvent(category, action, label, dimJson)
	u := graph.AnalyticsEventResponse{}
	if err := m.request(ctx, r, &u); err != nil {
		return errs.Wrap(err, "Error sending analytics event via state-svc")
	}

	return nil
}

func (m *SvcModel) RecordRuntimeUsage(ctx context.Context, pid int, exec string, dimJson string) error {
	defer profile.Measure("svc:analyticsEvent", time.Now())

	r := request.NewAnalyticsEvent(category, action, label, dimJson)
	u := graph.AnalyticsEventResponse{}
	if err := m.request(ctx, r, &u); err != nil {
		return errs.Wrap(err, "Error sending analytics event via state-svc")
	}

	return nil
}