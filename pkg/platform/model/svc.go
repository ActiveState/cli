package model

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/retryhttp"
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

func (m *SvcModel) InitiateDeferredUpdate(ctx context.Context, channel, version string) (*graph.DeferredUpdate, error) {
	r := request.NewUpdateRequest(channel, version)
	u := graph.UpdateResponse{}
	if err := m.client.RunWithContext(ctx, r, &u); err != nil {
		return nil, locale.WrapError(err, "err_svc_updaterequest", "Error updating to version {{.V0}} at channel {{.V1}}: {{.V2}}", version, channel, errs.Join(err, ": ").Error())
	}
	return &u.DeferredUpdate, nil
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

func (m *SvcModel) Quit(ctx context.Context) (chan bool, error) {
	response := graph.QuitResponse{}
	result := make(chan bool)
	_, err := m.client.Subscribe(&response, nil, func(message *json.RawMessage, err error) error {
		if err != nil {
			return nil
		}

		err = json.Unmarshal(*message, &response)
		result <- response.Quit
		return nil
	})
	if err != nil {
		return nil, errs.Wrap(err, "Could not subscribe")
	}

	go m.client.SubscriptionClient.Run()

	return result, nil
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
	_, err := m.StateVersion(context.Background())
	return err
}

func (m *SvcModel) CloseSubscriptions() error {
	return m.client.Close()
}
