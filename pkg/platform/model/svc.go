package model

import (
	"context"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/platform/api/svc"
	"github.com/ActiveState/cli/pkg/platform/api/svc/request"
)

type SvcModel struct {
	client *svc.Client
}

// NewSvcModel returns a model for all client connections to a State Svc.  This function returns an error if the State service is not yet ready to communicate.
func NewSvcModel(ctx context.Context, cfg *config.Instance, svcm *svcmanager.Manager) (*SvcModel, error) {
	defer profile.Measure("NewSvcModel", time.Now())

	if err := svcm.WaitWithContext(ctx); err != nil {
		return nil, errs.Wrap(err, "Failed to wait for svc connection to be ready")
	}

	client, err := svc.New(cfg)
	if err != nil {
		return nil, errs.Wrap(err, "Could not initialize svc client")
	}

	return newSvcModelWithClient(client), nil
}

func newSvcModelWithClient(client *svc.Client) *SvcModel {
	return &SvcModel{client}
}

func (m *SvcModel) StateVersion(ctx context.Context) (*graph.Version, error) {
	r := request.NewVersionRequest()
	resp := graph.VersionResponse{}
	if err := m.client.RunWithContext(ctx, r, &resp); err != nil {
		return nil, err
	}
	return &resp.Version, nil
}

func (m *SvcModel) LocalProjects(ctx context.Context) ([]*graph.Project, error) {
	r := request.NewLocalProjectsRequest()
	response := graph.ProjectsResponse{Projects: []*graph.Project{}}
	if err := m.client.RunWithContext(ctx, r, &response); err != nil {
		return nil, err
	}
	return response.Projects, nil
}

func (m *SvcModel) CheckUpdate(ctx context.Context) (*graph.AvailableUpdate, error) {
	defer profile.Measure("svc:CheckUpdate", time.Now())
	r := request.NewAvailableUpdate()
	u := graph.AvailableUpdateResponse{}
	if err := m.client.RunWithContext(ctx, r, &u); err != nil {
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

func (m *SvcModel) AnalyticsEventWithLabel(ctx context.Context, category, action, label, projectName, output, userID string) error {
	defer profile.Measure("svc:analyticsEvent", time.Now())

	r := request.NewAnalyticsEvent(category, action, label, projectName, output, userID)
	u := graph.AnalyticsEventResponse{}
	if err := m.client.RunWithContext(ctx, r, &u); err != nil {
		return errs.Wrap(err, "Error sending analytics event via state-svc")
	}

	return nil
}
