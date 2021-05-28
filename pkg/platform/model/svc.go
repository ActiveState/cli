package model

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/platform/api/svc"
	"github.com/ActiveState/cli/pkg/platform/api/svc/request"
)

type SvcModel struct {
	ctx    context.Context
	client *svc.Client
}

// NewSvcModel returns a model for all client connections to a State Svc.  This function returns an error if the State service is not yet ready to communicate.
func NewSvcModel(ctx context.Context, cfg *config.Instance, svcm *svcmanager.Manager) (*SvcModel, error) {
	client, err := svc.New(cfg)
	if err != nil {
		return nil, errs.Wrap(err, "Could not initialize svc client")
	}

	if err := svcm.Wait(pingFunction(cfg)); err != nil {
		return nil, errs.Wrap(err, "Failed to wait for svc connection to be ready")
	}

	return newSvcModelWithClient(ctx, client), nil
}

// pingFunction returns a function that pings the server without guarantee of succeeding or retying on failure
func pingFunction(cfg *config.Instance) func(context.Context) error {
	return func(ctx context.Context) error {
		client, err := svc.NewWithoutRetry(cfg)
		if err != nil {
			return errs.Wrap(err, "Could not initialize non-retrying svc client")
		}

		m := newSvcModelWithClient(ctx, client)
		return m.Ping()
	}
}

func newSvcModelWithClient(ctx context.Context, client *svc.Client) *SvcModel {
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

func (m *SvcModel) StopServer() error {
	htClient := retryhttp.DefaultClient.StandardClient()

	quitAddress := fmt.Sprintf("%s/__quit", m.client.BaseUrl())
	logging.Debug("Sending quit request to %s", quitAddress)
	req, err := http.NewRequest("GET", quitAddress, nil)
	if err != nil {
		return errs.Wrap(err, "Could not create request to quit svc")
	}

	res, err := htClient.Do(req)
	if err != nil {
		return errs.Wrap(err, "Request to quit svc failed")
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return errs.Wrap(err, "Request to quit svc responded with status %s", res.Status)
		}
		return errs.New("Request to quit svc responded with status: %s, response: %s", res.Status, body)
	}

	return nil
}

func (m *SvcModel) Ping() error {
	_, err := m.StateVersion()
	return err
}
