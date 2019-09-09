package headchef

import (
	"net/url"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_client"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_client/headchef_operations"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
)

var (
	FailRequestConnect = failures.Type("headchef.fail.request.connect", failures.FailNetwork)

	FailRequestMarshal = failures.Type("headchef.fail.request.marshal", failures.FailMarshal)

	FailRequestUnmarshal = failures.Type("headchef.fail.request.unmarshal", failures.FailMarshal)

	FailRequestUnmarshalStatus = failures.Type("headchef.fail.request.unmarshalstatus", FailRequestUnmarshal)

	FailRequestAtDisconnect = failures.Type("headchef.fail.request.atdisconnect")

	FailRequestValidation = failures.Type("headchef.fail.request.validation")

	FailRestAPIError = failures.Type("headchef.fail.restapi.error")
)

type Requester interface {
	OnBuildStarted(f RequestBuildStarted)
	OnBuildFailed(f RequestBuildFailed)
	OnBuildCompleted(f RequestBuildCompleted)
	OnFailure(f RequestFailure)
	OnClose(f RequestClose)
	Start()
}

type Request struct {
	buildRequest *headchef_models.V1BuildRequest
	client       *headchef_operations.Client

	onBuildStarted   RequestBuildStarted
	onBuildFailed    RequestBuildFailed
	onBuildCompleted RequestBuildCompleted
	onFailure        RequestFailure
	onClose          RequestClose
}

type RequestBuildStarted func()
type RequestBuildFailed func(message string)
type RequestBuildCompleted func(headchef_models.BuildCompletedResponse)
type RequestFailure func(*failures.Failure)
type RequestClose func()

type InitRequester func(buildRequest *headchef_models.V1BuildRequest) Requester

func InitRequest(buildRequest *headchef_models.V1BuildRequest) Requester {
	return NewRequest(api.GetServiceURL(api.ServiceHeadChef), buildRequest)
}

func NewRequest(u *url.URL, buildRequest *headchef_models.V1BuildRequest) *Request {
	return &Request{
		buildRequest: buildRequest,
		client:       headchef_client.Default.HeadchefOperations,
	}
}

func (r *Request) OnBuildStarted(f RequestBuildStarted) {
	r.onBuildStarted = f
}

func (r *Request) triggerBuildStarted() {
	logging.Debug("BuildStarted")
	if r.onBuildStarted != nil {
		r.onBuildStarted()
	}
}

func (r *Request) OnBuildFailed(f RequestBuildFailed) {
	r.onBuildFailed = f
}

func (r *Request) triggerBuildFailed(message string) {
	logging.Debug("BuildFailed: %s", message)
	if r.onBuildFailed != nil {
		r.onBuildFailed(message)
	}
}

func (r *Request) OnBuildCompleted(f RequestBuildCompleted) {
	r.onBuildCompleted = f
}

func (r *Request) triggerBuildCompleted(response headchef_models.BuildCompletedResponse) {
	logging.Debug("BuildCompleted:", response)
	if r.onBuildCompleted != nil {
		r.onBuildCompleted(response)
	}
}

func (r *Request) OnFailure(f RequestFailure) {
	r.onFailure = f
}

func (r *Request) triggerFailure(fail *failures.Failure) {
	logging.Debug("Failure: %v", fail)
	if r.onFailure != nil {
		r.onFailure(fail)
	}
}

func (r *Request) OnClose(f RequestClose) {
	r.onClose = f
}

func (r *Request) triggerClose() {
	logging.Debug("Close")
	if r.onClose != nil {
		r.onClose()
	}
}

func (r *Request) Start() {
	max := constants.HeadChefBuildStatusCheckMax
	eager := time.Minute * 3 // duration to use short polling wait duration
	shortWait := time.Second * 8
	longWait := time.Second * 16

	logging.Debug("connecting to head-chef")

	defer r.triggerClose()

	var buildUUID *strfmt.UUID

	startParams := headchef_operations.StartBuildV1Params{
		BuildRequest: r.buildRequest,
	}
	created, accepted, err := r.client.StartBuildV1(&startParams)
	switch {
	case err != nil:
		r.triggerFailure(FailRestAPIError.Wrap(err))
		return
	case accepted != nil:
		r.triggerBuildStarted()
		buildUUID = accepted.Payload.BuildRequestID
	case created != nil:
		switch payload := created.Payload.(type) {
		case headchef_models.BuildCompletedResponse:
			r.triggerBuildCompleted(payload)
			return
		case headchef_models.BuildFailedResponse:
			r.triggerBuildFailed(payload.Message)
			return
		default:
			logging.Panic("unknown BuildEndedResponse payload type")
		}
	default:
		logging.Panic("no value returned from StartBuildV1")
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
		buildStatus, err := r.client.GetBuildStatus(&buildStatusParams)
		if err != nil {
			r.triggerFailure(FailRestAPIError.Wrap(err))
			return
		}
		switch payload := buildStatus.Payload.(type) {
		case headchef_models.BuildStartedResponse:
			continue
		case headchef_models.BuildCompletedResponse:
			r.triggerBuildCompleted(payload)
			return
		case headchef_models.BuildFailedResponse:
			r.triggerBuildFailed(payload.Message)
			return
		default:
			logging.Panic("unknown BuildRequestedResponse type")
		}
	}

	logging.Error(locale.T("build_status_timeout"))
}
