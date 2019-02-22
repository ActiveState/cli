package headchef

import (
	"encoding/json"
	"net/url"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/google/uuid"
	"github.com/sacOO7/gowebsocket"
)

var DefaultDialer *websocket.Dialer

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

func InitRequest(recipe *headchef_models.BuildRequestRecipe, requestor *headchef_models.BuildRequestRequester) *Request {
	return NewRequest(api.GetServiceURL(api.ServiceHeadChef), recipe, requestor, DefaultDialer)
}

func NewRequest(u *url.URL, recipe *headchef_models.BuildRequestRecipe, requestor *headchef_models.BuildRequestRequester, dialer *websocket.Dialer) *Request {
	logging.Debug("connecting to head-chef at %s", u.String())

	socket := gowebsocket.New(u.String())
	if dialer != nil {
		socket.WebsocketDialer = dialer
	}
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

func (r *Request) close() {
	// Work around strange bug where socket.Close() times out on writing the close message.
	// Oddly the timeout imposed close call works just fine.
	// I've only encountered this issue in tests, so might be an issue with our testing library
	r.socket.Conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	go r.socket.Close() // run in subroutine so the close doesn't block anything, it's of no consequence
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
	if envelope.Type == nil {
		return false // this isn't a status message
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
		r.close()
		return true

	// Build Failed
	case headchef_models.StatusMessageEnvelopeTypeBuildFailed:
		response := headchef_models.BuildFailed{}
		json, err := json.Marshal(envelope.Body)
		if err != nil {
			r.triggerFailure(FailRequestUnmarshalStatus.Wrap(err))
			return true
		}
		err = response.UnmarshalBinary(json)
		if err != nil {
			r.triggerFailure(FailRequestUnmarshalStatus.Wrap(err))
		} else {
			logging.Warning("head-chef build failed with the following errors: %v", response.Errors)
			r.triggerBuildFailed(response.Message)
		}
		r.close()
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
		// The error here is useless, because gowebsocket just forwards the close message as an error, regardless of whether
		// it actually is one
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
