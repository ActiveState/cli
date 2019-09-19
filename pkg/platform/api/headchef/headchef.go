package headchef

import (
	"context"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_client"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_client/headchef_operations"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
)

var (
	FailRestAPIError = failures.Type("headchef.fail.restapi.error")
)

type BuildStatus struct {
	Started   chan struct{}
	Failed    chan string
	Completed chan headchef_models.BuildCompletedResponse
	RunFail   chan *failures.Failure
}

func NewBuildStatus() *BuildStatus {
	return &BuildStatus{
		Started:   make(chan struct{}),
		Failed:    make(chan string),
		Completed: make(chan headchef_models.BuildCompletedResponse),
		RunFail:   make(chan *failures.Failure),
	}
}

func (s *BuildStatus) Close() {
	close(s.Started)
	close(s.Failed)
	close(s.Completed)
	close(s.RunFail)
}

type Request struct {
	client *headchef_operations.Client
}

func InitRequest() *Request {
	return NewRequest(api.GetSettings(api.ServiceHeadChef))
}

func NewRequest(apiSetting api.Settings) *Request {
	transportRuntime := httptransport.New(apiSetting.Host, apiSetting.BasePath, []string{apiSetting.Schema})
	transportRuntime.Transport = api.NewUserAgentTripper()

	transportRuntime.SetDebug(true)

	return &Request{
		client: headchef_client.New(transportRuntime, strfmt.Default).HeadchefOperations,
	}
}

func (r *Request) Run(buildRequest *headchef_models.V1BuildRequest) *BuildStatus {
	buildStatus := NewBuildStatus()

	go r.run(buildRequest, buildStatus)

	return buildStatus
}

func (r *Request) run(buildRequest *headchef_models.V1BuildRequest, buildStatus *BuildStatus) {
	defer buildStatus.Close()

	startParams := headchef_operations.StartBuildV1Params{
		Context:      context.Background(),
		BuildRequest: buildRequest,
	}
	created, accepted, err := r.client.StartBuildV1(&startParams)

	switch {
	case err != nil:
		if startErr, ok := err.(*headchef_operations.StartBuildV1Default); ok {
			msg := *startErr.Payload.Message
			buildStatus.RunFail <- FailRestAPIError.New(msg)
			return
		}
		buildStatus.RunFail <- FailRestAPIError.Wrap(err)
		return
	case accepted != nil:
		buildStatus.RunFail <- FailRestAPIError.New(locale.T("build_status_in_progress"))
	case created != nil:
		switch payload := created.Payload.(type) {
		case headchef_models.BuildCompletedResponse:
			buildStatus.Completed <- payload
			return
		case headchef_models.BuildFailedResponse:
			buildStatus.Failed <- payload.Message
			return
		// Go swagger is NUTS, it cannot generate BuildFailedResponse
		case map[string]interface{}:
			buildStatus.Failed <- payload["message"].(string)
		case headchef_models.BuildEndedResponse:
			if p, ok := payload.(headchef_models.BuildFailedResponse); ok {
				buildStatus.Failed <- p.Message
			}
			buildStatus.Failed <- locale.T("build_status_unknown_end")
			return
		case headchef_models.BuildStartedResponse:
		case headchef_models.BuildStarted:
			buildStatus.Failed <- locale.T("build_status_in_progress")
			return
		case headchef_models.RestAPIError:
			if payload.Message == nil {
				buildStatus.Failed <- locale.T("build_status_unknown_error")
			} else {
				buildStatus.Failed <- *payload.Message
			}
		default:
			buildStatus.Failed <- locale.T("build_status_unknown_status")
			return
		}
	default:
		buildStatus.RunFail <- FailRestAPIError.New(locale.T("build_status_noresponse"))
		return
	}
}
