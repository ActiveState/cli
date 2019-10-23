package headchef

import (
	"context"
	"net/url"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_client"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_client/headchef_operations"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
)

var (
	FailRestAPIError       = failures.Type("headchef.fail.restapi.error")
	FailRestAPINoResponse  = failures.Type("headchef.fail.restapi.noresponse")
	FailRestAPIBadResponse = failures.Type("headchef.fail.restapi.badresponse")
)

type BuildStatus struct {
	Started   chan struct{}
	Failed    chan string
	Completed chan *headchef_models.BuildStatusResponse
	RunFail   chan *failures.Failure
}

func NewBuildStatus() *BuildStatus {
	return &BuildStatus{
		Started:   make(chan struct{}),
		Failed:    make(chan string),
		Completed: make(chan *headchef_models.BuildStatusResponse),
		RunFail:   make(chan *failures.Failure),
	}
}

func (s *BuildStatus) Close() {
	close(s.Started)
	close(s.Failed)
	close(s.Completed)
	close(s.RunFail)
}

type BuildStatusClient struct {
	client *headchef_operations.Client
}

func InitBuildStatusClient() *BuildStatusClient {
	return NewBuildStatusClient(api.GetServiceURL(api.ServiceHeadChef))
}

func NewBuildStatusClient(apiURL *url.URL) *BuildStatusClient {
	transportRuntime := httptransport.New(apiURL.Host, apiURL.Path, []string{apiURL.Scheme})
	transportRuntime.Transport = api.NewUserAgentTripper()

	//transportRuntime.SetDebug(true)

	return &BuildStatusClient{
		client: headchef_client.New(transportRuntime, strfmt.Default).HeadchefOperations,
	}
}

func (r *BuildStatusClient) Run(buildRequest *headchef_models.V1BuildRequest) *BuildStatus {
	buildStatus := NewBuildStatus()

	go func() {
		defer buildStatus.Close()
		r.run(buildRequest, buildStatus)
	}()

	return buildStatus
}

func (r *BuildStatusClient) run(buildRequest *headchef_models.V1BuildRequest, buildStatus *BuildStatus) {
	startParams := headchef_operations.StartBuildV1Params{
		Context:      context.Background(),
		BuildRequest: buildRequest,
	}
	created, accepted, err := r.client.StartBuildV1(&startParams)

	switch {
	case err != nil:
		msg := err.Error()
		if startErr, ok := err.(*headchef_operations.StartBuildV1Default); ok {
			msg = *startErr.Payload.Message
		}
		buildStatus.RunFail <- FailRestAPIError.New(msg)
	case accepted != nil:
		buildStatus.Started <- struct{}{}
	case created != nil:
		if created.Payload.Type == nil {
			junk := "junk"
			created.Payload.Type = &junk
		}

		switch *created.Payload.Type {
		case headchef_models.BuildStatusResponseTypeBuildCompleted:
			buildStatus.Completed <- created.Payload
		case headchef_models.BuildStatusResponseTypeBuildFailed:
			buildStatus.Failed <- created.Payload.Message
		case headchef_models.BuildStatusResponseTypeBuildStarted:
			buildStatus.Started <- struct{}{}
		default:
			buildStatus.RunFail <- FailRestAPIBadResponse.New("bad response")
		}
	default:
		buildStatus.RunFail <- FailRestAPINoResponse.New("no response")
	}
}
