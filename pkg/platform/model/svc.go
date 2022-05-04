package model

import (
	"context"
	"net/http"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/pkg/platform/api/svc/request"
	"github.com/machinebox/graphql"
)

var (
	SvcTimeoutMinimal = time.Millisecond * 500
)

type SvcModel struct {
	client *gqlclient.Client
}

// NewSvcModel returns a model for all client connections to a State Svc.  This function returns an error if the State service is not yet ready to communicate.
func NewSvcModel(port string) *SvcModel {
	localURL := "http://127.0.0.1" + port + "/query"

	return &SvcModel{
		client: gqlclient.NewWithOpts(localURL, 0, graphql.WithHTTPClient(&http.Client{})),
	}
}

func (m *SvcModel) request(ctx context.Context, request gqlclient.Request, resp interface{}) error {
	defer profile.Measure("SvcModel:request", time.Now())
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

	// TODO: https://activestatef.atlassian.net/browse/DX-866
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
	defer profile.Measure("svc:RecordRuntimeUsage", time.Now())

	r := request.NewRuntimeUsage(pid, exec, dimJson)
	u := graph.RuntimeUsageResponse{}
	if err := m.request(ctx, r, &u); err != nil {
		return errs.Wrap(err, "Error sending runtime usage event via state-svc")
	}

	return nil
}

func (m *SvcModel) CheckDeprecation(ctx context.Context) (*graph.DeprecationInfo, error) {
	logging.Debug("Checking for deprecation")
	defer profile.Measure("svc:CheckDeprecation", time.Now())

	r := request.NewDeprecationRequest()
	uu := graph.DeprecationResponse{}
	if err := m.request(ctx, r, &uu); err != nil {
		return nil, errs.Wrap(err, "Error sending deprecation request")
	}

	u := uu.CheckDeprecation
	// TODO: https://activestatef.atlassian.net/browse/DX-866
	if u.Date == "" {
		return nil, nil
	}

	return &u, nil
}

func (m *SvcModel) ConfigChanged(ctx context.Context, key string) error {
	defer profile.Measure("svc:RecordRuntimeUsage", time.Now())

	r := request.NewConfigChanged(key)
	u := graph.ConfigChangedResponse{}
	if err := m.request(ctx, r, &u); err != nil {
		return errs.Wrap(err, "Error sending configchanged event via state-svc")
	}

	return nil
}
