package headchef

import (
	"encoding/json"
	"net/url"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/google/uuid"
	"github.com/sacOO7/gowebsocket"
)

var test string = `{"build_request_id":"f4cb6b30-fd90-5b01-a297-b3111e0706a4","recipe":{"platform_id":"eef02e93-f4a9-5cca-a131-a388ecf57442","recipe_id":"f4cb6b30-fd90-5b01-a297-b3111e0706a4","resolved_requirements":[{"ingredient":{"ingredient_id":"1bf08759-25a9-5b49-8008-9a370634404e","name":"ActivePythonEnterprise-3.5.4.3504-source-626c6001.zip","description":"","namespace":"pre-platform-installer"},"ingredient_version":{"description":"pre-platform installer for ActivePythonEnterprise-3.5.4.3504-source-626c6001.zip","ingredient_id":"1bf08759-25a9-5b49-8008-9a370634404e","ingredient_version_id":"7f255b06-b74c-5496-94e0-db3eb698f884","is_stable_release":true,"release_date":"1970-01-01T00:00:00.000Z","revision":1,"source_uri":"https://www.activestate.com/","version":"1"}},{"ingredient":{"ingredient_id":"85374083-8434-59fe-8e7c-54e46dbda8af","name":"ActivePythonEnterprise-3.6.6.3606-source-63c815b2.tar.gz","description":"","namespace":"pre-platform-installer"},"ingredient_version":{"description":"pre-platform installer for ActivePythonEnterprise-3.6.6.3606-source-63c815b2.tar.gz","ingredient_id":"85374083-8434-59fe-8e7c-54e46dbda8af","ingredient_version_id":"67f71792-34b4-5400-b606-64afc253ab53","is_stable_release":true,"release_date":"1970-01-01T00:00:00.000Z","revision":1,"source_uri":"https://www.activestate.com/","version":"1"}}]},"requester":{"project_id":"9082cbea-2938-413a-8b1c-2188b578f5ce","organization_id":"2b53beaa-5189-4358-b980-ce236a5269b4","user_id":"7a481c85-5521-4899-82fb-71bae071c486"}}`

var (
	FailRequestConnect = failures.Type("headchef.fail.request.connect", failures.FailNetwork)

	FailRequestMarshal = failures.Type("headchef.fail.request.marshal", failures.FailMarshal)

	FailRequestUnmarshal = failures.Type("headchef.fail.request.unmarshal", failures.FailMarshal)

	FailRequestUnmarshalStatus = failures.Type("headchef.fail.request.unmarshalstatus", FailRequestUnmarshal)

	FailRequestAtDisconnect = failures.Type("headchef.fail.request.atdisconnect")

	FailRequestValidation = failures.Type("headchef.fail.request.validation")
)

type Request struct {
	socket    gowebsocket.Socket
	recipe    *headchef_models.BuildRequestRecipe
	requestor *headchef_models.BuildRequestRequester

	onBuildStarted   RequestBuildStarted
	onBuildFailed    RequestBuildFailed
	onBuildCompleted RequestBuildCompleted
	onFailure        RequestFailure
	onClose          RequestClose
}

type RequestBuildStarted func()
type RequestBuildFailed func(message string)
type RequestBuildCompleted func(headchef_models.BuildCompleted)
type RequestFailure func(*failures.Failure)
type RequestClose func()

func NewRequest(recipe *headchef_models.BuildRequestRecipe, requestor *headchef_models.BuildRequestRequester) *Request {
	return InitRequest(api.GetServiceURL(api.ServiceHeadChef), recipe, requestor)
}

func InitRequest(u *url.URL, recipe *headchef_models.BuildRequestRecipe, requestor *headchef_models.BuildRequestRequester) *Request {
	logging.Debug("connecting to head-chef at %s", u.String())

	socket := gowebsocket.New(u.String())
	socket.RequestHeader.Set("Origin", constants.HeadChefOrigin)

	request := &Request{socket: socket, recipe: recipe, requestor: requestor}

	return request
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

func (r *Request) triggerBuildCompleted(response headchef_models.BuildCompleted) {
	logging.Debug("BuildCompleted:", response.Message)
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

func (r *Request) handleMessage(message string, socket gowebsocket.Socket) {
	if r.handleValidationError(message) {
		return
	}

	if !r.handleStatusMessage(message) {
		// If neither handleValidationError nor handleStatusMessage handled the message we potentially have a problem
		// Though it could be nothing (eg. the headchef was updated with new messages that aren't required for backwards compatibility)
		logging.Warning("Unrecognized message: %s", message)
	}
}

func (r *Request) handleValidationError(message string) (handled bool) {
	validationError := headchef_models.RestAPIValidationError{}
	err := validationError.UnmarshalJSON([]byte(message))
	if err == nil && validationError.ValidationErrors != nil {
		errMsg := message
		if validationError.Message != nil {
			errMsg = *validationError.Message
		}
		r.triggerFailure(FailRequestValidation.New(errMsg))
		r.socket.Close()
		return true
	}

	return false
}

func (r *Request) handleStatusMessage(message string) (handled bool) {
	envelope := headchef_models.StatusMessageEnvelope{}
	err := envelope.UnmarshalBinary([]byte(message))
	if err != nil {
		logging.Error("Could not unmarshal websocket response, error: %v", err)
		return false
	}

	switch *envelope.Type {
	// Build Started
	case headchef_models.StatusMessageEnvelopeTypeBuildStarted:
		r.triggerBuildStarted()
		return true

	// Build Completed
	case headchef_models.StatusMessageEnvelopeTypeBuildCompleted:
		response := headchef_models.BuildCompleted{}
		json, err := json.Marshal(envelope.Body)
		if err != nil {
			r.triggerFailure(FailRequestUnmarshalStatus.Wrap(err))
			return true
		}
		err = response.UnmarshalBinary(json)
		if err != nil {
			r.triggerFailure(FailRequestUnmarshalStatus.Wrap(err))
		} else {
			r.triggerBuildCompleted(response)
		}
		r.socket.Close()
		return true

	// Build Failed
	case headchef_models.StatusMessageEnvelopeTypeBuildFailed:
		response := headchef_models.BuildFailed{}
		err := response.UnmarshalBinary([]byte(envelope.Body.(string)))
		if err != nil {
			r.triggerFailure(FailRequestUnmarshalStatus.Wrap(err))
		} else {
			logging.Warning("head-chef build failed with the following errors: %v", response.Errors)
			r.triggerBuildFailed(response.Message)
		}
		r.socket.Close()
		return true
	}

	return false
}

func (r *Request) Start() {
	// Hook up our event handlers
	r.socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		r.triggerFailure(FailRequestConnect.Wrap(err))
	}
	r.socket.OnTextMessage = r.handleMessage
	r.socket.OnDisconnected = func(err error, socket gowebsocket.Socket) {
		if err != nil {
			r.triggerFailure(FailRequestAtDisconnect.Wrap(err))
		}
		r.triggerClose()
	}

	r.socket.OnConnected = func(socket gowebsocket.Socket) {
		logging.Debug("Connected")

		// Send our build request
		uuid := strfmt.UUID(uuid.New().String())
		buildRequest := headchef_models.BuildRequest{
			BuildRequestID: &uuid,
			Recipe:         r.recipe,
			Requester:      r.requestor,
		}
		bytes, err := buildRequest.MarshalBinary()
		if err != nil {
			r.triggerFailure(FailRequestMarshal.Wrap(err))
		}
		r.socket.SendBinary(bytes)
	}

	r.socket.Connect()
}
