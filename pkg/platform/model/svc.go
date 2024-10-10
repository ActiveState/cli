package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/api/svc/request"
	"github.com/ActiveState/graphql"
)

var SvcTimeoutMinimal = time.Millisecond * 500

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

// EnableDebugLog turns on debug logging
func (m *SvcModel) EnableDebugLog() {
	m.client.EnableDebugLog()
}

func (m *SvcModel) request(ctx context.Context, request gqlclient.Request, resp interface{}) error {
	defer profile.Measure("SvcModel:request", time.Now())

	err := m.client.RunWithContext(ctx, request, resp)
	if err != nil {
		reqError := &gqlclient.RequestError{}
		if errors.As(err, &reqError) && (!condition.BuiltViaCI() || condition.InTest()) {
			vars, err := request.Vars()
			if err != nil {
				return errs.Wrap(err, "Could not get variables")
			}
			logging.Debug(
				"svc client gql request failed - query: %q, vars: %q",
				reqError.Request.Query(),
				jsonFromMap(vars),
			)
		}
		return err
	}

	return nil
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

// CheckUpdate returns cached update information. There is no guarantee that
// available information is immediately cached. For instance, if this info is
// requested shortly after the service is started up, the data may return
// empty for a little while.
func (m *SvcModel) CheckUpdate(ctx context.Context, desiredChannel, desiredVersion string) (*graph.AvailableUpdate, error) {
	defer profile.Measure("svc:CheckUpdate", time.Now())
	r := request.NewAvailableUpdate(desiredChannel, desiredVersion)
	u := graph.AvailableUpdateResponse{}
	if err := m.request(ctx, r, &u); err != nil {
		return nil, errs.Wrap(err, "Error checking if update is available.")
	}

	return &u.AvailableUpdate, nil
}

func (m *SvcModel) Ping() error {
	_, err := m.StateVersion(context.Background())
	return err
}

func (m *SvcModel) AnalyticsEvent(ctx context.Context, category, action, source, label string, dimJson string) error {
	defer profile.Measure("svc:analyticsEvent", time.Now())

	r := request.NewAnalyticsEvent(category, action, source, label, dimJson)
	u := graph.AnalyticsEventResponse{}
	if err := m.request(ctx, r, &u); err != nil {
		return errs.Wrap(err, "Error sending analytics event via state-svc")
	}

	return nil
}

func (m *SvcModel) ReportRuntimeUsage(ctx context.Context, pid int, exec, source string, dimJson string) error {
	defer profile.Measure("svc:ReportRuntimeUsage", time.Now())

	r := request.NewReportRuntimeUsage(pid, exec, source, dimJson)
	u := graph.ReportRuntimeUsageResponse{}
	if err := m.request(ctx, r, &u); err != nil {
		return errs.Wrap(err, "Error sending report runtime usage event via state-svc")
	}

	return nil
}

func (m *SvcModel) CheckMessages(ctx context.Context, command string, flags []string) ([]*graph.MessageInfo, error) {
	logging.Debug("Checking for messages")
	defer profile.Measure("svc:CheckMessages", time.Now())

	r := request.NewMessagingRequest(command, flags)
	resp := graph.CheckMessagesResponse{}
	if err := m.request(ctx, r, &resp); err != nil {
		return nil, errs.Wrap(err, "Error sending messages request")
	}

	return resp.Messages, nil
}

func (m *SvcModel) ConfigChanged(ctx context.Context, key string) error {
	defer profile.Measure("svc:ConfigChanged", time.Now())

	r := request.NewConfigChanged(key)
	u := graph.ConfigChangedResponse{}
	if err := m.request(ctx, r, &u); err != nil {
		return errs.Wrap(err, "Error sending configchanged event via state-svc")
	}

	return nil
}

func (m *SvcModel) FetchLogTail(ctx context.Context) (string, error) {
	logging.Debug("Fetching log svc log")
	defer profile.Measure("svc:FetchLogTail", time.Now())

	req := request.NewFetchLogTail()
	response := make(map[string]string)
	if err := m.request(ctx, req, &response); err != nil {
		return "", errs.Wrap(err, "Error sending FetchLogTail request to state-svc")
	}
	if log, ok := response["fetchLogTail"]; ok {
		return log, nil
	}
	return "", errs.New("svcModel.FetchLogTail() did not return an expected value")
}

func (m *SvcModel) GetProcessesInUse(ctx context.Context, execDir string) ([]*graph.ProcessInfo, error) {
	logging.Debug("Checking if runtime is in use for %s", execDir)
	defer profile.Measure("svc:GetProcessesInUse", time.Now())

	req := request.NewGetProcessesInUse(execDir)
	response := graph.GetProcessesInUseResponse{}
	if err := m.request(ctx, req, &response); err != nil {
		return nil, errs.Wrap(err, "Error sending GetProcessesInUse request to state-svc")
	}

	return response.Processes, nil
}

// GetJWT grabs the JWT from the svc, if it exists.
// Note we respond with mono_models.JWT here for compatibility and to minimize the changeset at time of implementation.
// We can revisit this in the future.
func (m *SvcModel) GetJWT(ctx context.Context) (*mono_models.JWT, error) {
	logging.Debug("Checking for GetJWT")
	defer profile.Measure("svc:GetJWT", time.Now())

	r := request.NewJWTRequest()
	resp := graph.GetJWTResponse{}
	if err := m.request(ctx, r, &resp); err != nil {
		return nil, errs.Wrap(err, "Error sending messages request")
	}

	jwt := &mono_models.JWT{}
	err := json.Unmarshal(resp.Payload, &jwt)
	if err != nil {
		return nil, errs.Wrap(err, "Error unmarshaling JWT")
	}

	return jwt, nil
}

func (m *SvcModel) GetCache(key string) (result string, _ error) {
	defer func() { logging.Debug("GetCache %s, result size: %d", key, len(result)) }()
	defer profile.Measure("svc:GetCache", time.Now())

	req := request.NewGetCache(key)
	response := make(map[string]string)
	if err := m.request(context.Background(), req, &response); err != nil {
		return "", errs.Wrap(err, "Error sending GetCache request to state-svc")
	}
	if entry, ok := response["getCache"]; ok {
		return entry, nil
	}
	return "", errs.New("svcModel.GetCache() did not return an expected value")
}

func (m *SvcModel) SetCache(key, value string, expiry time.Duration) error {
	logging.Debug("SetCache %s, value size: %d", key, len(value))
	defer profile.Measure("svc:SetCache", time.Now())

	req := request.NewSetCache(key, value, expiry)
	if err := m.request(context.Background(), req, ptr.To(make(map[string]string))); err != nil {
		return errs.Wrap(err, "Error sending SetCache request to state-svc")
	}
	return nil
}

func jsonFromMap(m map[string]interface{}) string {
	d, err := json.Marshal(m)
	if err != nil {
		return fmt.Sprintf("cannot marshal map (%q) as json: %v", stringFromMap(m), err)
	}
	return string(d)
}

func stringFromMap(m map[string]interface{}) string {
	var s, sep string
	for k, v := range m {
		s += fmt.Sprintf("%s%s: %#v", sep, k, v)
		sep = ", "
	}
	return s
}
