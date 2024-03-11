package headchef

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_client"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_client/headchef_operations"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var (
	ErrBuildResp        = errs.New("Build responded with error")
	ErrBuildUnknownType = errs.New("Unknown build type")
)

type BuildStatus struct {
	Started   chan *headchef_models.V1BuildStatusResponse
	Failed    chan string
	Completed chan *headchef_models.V1BuildStatusResponse
	RunError  chan error
}

type BuildAnnotations struct {
	CommitID     string `json:"commit_id"`
	Project      string `json:"project"`
	Organization string `json:"organization"`
}

func NewBuildStatus() *BuildStatus {
	return &BuildStatus{
		Started:   make(chan *headchef_models.V1BuildStatusResponse),
		Failed:    make(chan string),
		Completed: make(chan *headchef_models.V1BuildStatusResponse),
		RunError:  make(chan error),
	}
}

func (s *BuildStatus) Close() {
	close(s.Started)
	close(s.Failed)
	close(s.Completed)
	close(s.RunError)
}

type Client struct {
	client    headchef_operations.Client
	transport *httptransport.Runtime
	auth      *authentication.Auth
}

func InitClient(auth *authentication.Auth) *Client {
	return NewClient(api.GetServiceURL(api.ServiceHeadChef), auth)
}

func NewClient(apiURL *url.URL, auth *authentication.Auth) *Client {
	logging.Debug("apiURL: %s", apiURL.String())
	transportRuntime := httptransport.New(apiURL.Host, apiURL.Path, []string{apiURL.Scheme})
	transportRuntime.Transport = api.NewRoundTripper(http.DefaultTransport)

	// transportRuntime.SetDebug(true)

	if auth != nil {
		transportRuntime.DefaultAuthentication = auth.ClientAuth()
	}

	return &Client{
		client:    *headchef_client.New(transportRuntime, strfmt.Default).HeadchefOperations,
		transport: transportRuntime,
		auth:      auth,
	}
}

func (r *Client) RequestBuild(buildRequest *headchef_models.V1BuildRequest) *BuildStatus {
	buildStatus := NewBuildStatus()

	go func() {
		defer buildStatus.Close()
		r.reqBuild(buildRequest, buildStatus)
	}()

	return buildStatus
}

func (r *Client) RequestBuildSync(buildRequest *headchef_models.V1BuildRequest) (BuildStatusEnum, *headchef_models.V1BuildStatusResponse, error) {
	return r.reqBuildSync(buildRequest)
}

func NewBuildRequest(recipeID, orgID, projID strfmt.UUID, annotations BuildAnnotations) (*headchef_models.V1BuildRequest, error) {
	uid := strfmt.UUID("00010001-0001-0001-0001-000100010001")

	br := &headchef_models.V1BuildRequest{
		Requester: &headchef_models.V1BuildRequestRequester{
			OrganizationID: &orgID,
			ProjectID:      &projID,
			UserID:         uid,
		},
		RecipeID:    recipeID,
		Annotations: annotations,
	}

	return br, nil
}

type BuildParams struct {
	headchef_operations.StartBuildV1Params
	timeout      time.Duration
	BuildRequest *headchef_models.V1BuildRequest
}

func (b *BuildParams) WithTimeout(timeout time.Duration) *BuildParams {
	b.StartBuildV1Params.SetTimeout(timeout)
	return b
}

func (b *BuildParams) SetTimeout(timeout time.Duration) {
	b.timeout = timeout
}

func (b *BuildParams) WriteToRequest(req runtime.ClientRequest, reg strfmt.Registry) error {
	if err := req.SetTimeout(b.timeout); err != nil {
		return err
	}

	if b.BuildRequest != nil {
		if err := req.SetBodyParam(b.BuildRequest); err != nil {
			return err
		}
	}

	return nil
}

type BuildStatusEnum int

const (
	Accepted BuildStatusEnum = iota
	Started
	Completed
	Failed
	Error
)

func (r *Client) reqBuildSync(buildReq *headchef_models.V1BuildRequest) (BuildStatusEnum, *headchef_models.V1BuildStatusResponse, error) {
	startParams := headchef_operations.StartBuildV1Params{
		Context:      context.Background(),
		BuildRequest: buildReq,
		HTTPClient:   api.NewHTTPClient(),
	}

	created, accepted, err := r.client.StartBuildV1(&startParams, r.auth.ClientAuth())

	switch {
	case err != nil:
		msg := err.Error()
		if startErr, ok := err.(*headchef_operations.StartBuildV1Default); ok {
			msg = *startErr.Payload.Message
		}
		return Error, nil, errs.Wrap(ErrBuildResp, msg)
	case accepted != nil:
		return Accepted, accepted.Payload, nil
	case created != nil:
		if created.Payload.Type == nil {
			requestBytes, err := buildReq.MarshalBinary()
			if err != nil {
				requestBytes = []byte(
					fmt.Sprintf("cannot marshal request: %v", err),
				)
			}
			msg := fmt.Sprintf(
				"created response cannot be handled: nil type from request %q",
				string(requestBytes),
			)
			return Error, nil, errs.New("Payload type was nil, message: %s", msg)
		}
		payloadType := *created.Payload.Type

		switch payloadType {
		case headchef_models.V1BuildStatusResponseTypeBuildCompleted:
			return Completed, created.Payload, nil
		case headchef_models.V1BuildStatusResponseTypeBuildFailed:
			return Failed, created.Payload, nil
		case headchef_models.V1BuildStatusResponseTypeBuildStarted:
			return Started, created.Payload, nil
		default:
			msg := fmt.Sprintf(
				"created response cannot be handled: unknown type %q",
				payloadType,
			)
			return Error, nil, errs.Wrap(ErrBuildUnknownType, msg)
		}
	default:
		return Error, nil, errs.New("no response")
	}
}

func (r *Client) reqBuild(buildReq *headchef_models.V1BuildRequest, buildStatus *BuildStatus) {
	startParams := headchef_operations.StartBuildV1Params{
		Context:      context.Background(),
		BuildRequest: buildReq,
		HTTPClient:   api.NewHTTPClient(),
	}

	created, accepted, err := r.client.StartBuildV1(&startParams, r.auth.ClientAuth())

	switch {
	case err != nil:
		msg := err.Error()
		if startErr, ok := err.(*headchef_operations.StartBuildV1Default); ok {
			msg = *startErr.Payload.Message
		}
		buildStatus.RunError <- locale.WrapError(ErrBuildResp, msg)
	case accepted != nil:
		buildStatus.Started <- accepted.Payload
	case created != nil:
		if created.Payload.Type == nil {
			requestBytes, err := buildReq.MarshalBinary()
			if err != nil {
				requestBytes = []byte(
					fmt.Sprintf("cannot marshal request: %v", err),
				)
			}
			msg := fmt.Sprintf(
				"created response cannot be handled: nil type from request %q",
				string(requestBytes),
			)
			buildStatus.RunError <- errs.New("Payload type was nil, message: %s", msg)
			break
		}
		payloadType := *created.Payload.Type

		switch payloadType {
		case headchef_models.V1BuildStatusResponseTypeBuildCompleted:
			buildStatus.Completed <- created.Payload
		case headchef_models.V1BuildStatusResponseTypeBuildFailed:
			buildStatus.Failed <- created.Payload.Message
		case headchef_models.V1BuildStatusResponseTypeBuildStarted:
			buildStatus.Started <- created.Payload
		default:
			msg := fmt.Sprintf(
				"created response cannot be handled: unknown type %q",
				payloadType,
			)
			buildStatus.RunError <- locale.WrapError(ErrBuildUnknownType, msg)
		}
	default:
		buildStatus.RunError <- errs.New("no response")
	}
}
