package headchef

import (
	"net/url"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
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
	return NewRequest(api.GetServiceURL(api.ServiceHeadChef))
}

func NewRequest(u *url.URL) *Request {
	return &Request{
		client: headchef_client.Default.HeadchefOperations,
	}
}

func (r *Request) Run(buildRequest *headchef_models.V1BuildRequest) *BuildStatus {
	max := constants.HeadChefBuildStatusCheckMax
	eager := time.Minute * 3 // duration to use short polling wait duration
	shortWait := time.Second * 8
	longWait := time.Second * 16
	buildStatus := NewBuildStatus()

	go func() {
		defer buildStatus.Close()

		var buildUUID *strfmt.UUID

		startParams := headchef_operations.StartBuildV1Params{
			BuildRequest: buildRequest,
		}
		created, accepted, err := r.client.StartBuildV1(&startParams)
		switch {
		case err != nil:
			buildStatus.RunFail <- FailRestAPIError.Wrap(err)
			return
		case accepted != nil:
			buildStatus.Started <- struct{}{}
			buildUUID = accepted.Payload.BuildRequestID
		case created != nil:
			switch payload := created.Payload.(type) {
			case headchef_models.BuildCompletedResponse:
				buildStatus.Completed <- payload
				return
			case headchef_models.BuildFailedResponse:
				buildStatus.Failed <- payload.Message
				return
			default:
				buildStatus.RunFail <- FailRestAPIError.New("unknown BuildEndedResponse payload type") // l10n
				return
			}
		default:
			buildStatus.RunFail <- FailRestAPIError.New("no value returned from StartBuildV1") // l10n
			return
		}

		var wait time.Duration
		for start := time.Now(); time.Now().Sub(start) < max; {
			time.Sleep(wait)
			wait = shortWait
			if time.Now().Sub(start) > eager {
				wait = longWait
			}

			buildStatusParams := headchef_operations.GetBuildStatusParams{
				BuildRequestID: *buildUUID,
			}
			buildStatusEnvelope, err := r.client.GetBuildStatus(&buildStatusParams)
			if err != nil {
				buildStatus.RunFail <- FailRestAPIError.Wrap(err)
				return
			}
			switch payload := buildStatusEnvelope.Payload.(type) {
			case headchef_models.BuildStartedResponse:
				continue
			case headchef_models.BuildCompletedResponse:
				buildStatus.Completed <- payload
				return
			case headchef_models.BuildFailedResponse:
				buildStatus.Failed <- payload.Message
				return
			default:
				buildStatus.RunFail <- FailRestAPIError.New("unknown BuildRequestedResponse type") // l10n
				return
			}
		}

		buildStatus.RunFail <- FailRestAPIError.New(locale.T("build_status_timeout"))
	}()

	return buildStatus
}
