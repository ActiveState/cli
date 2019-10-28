package headchef

import (
	"context"
	"fmt"
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
	FailBuildReqErrorResp       = failures.Type("headchef.fail.buildreq.errorresp")
	FailBuildReqNoResp          = failures.Type("headchef.fail.buildreq.noresp")
	FailBuildCreatedUnknownType = failures.Type("headchef.fail.buildcreated.unknowntype")
	FailBuildCreatedNilType     = failures.Type("headchef.fail.buildcreated.niltype")
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

type Client struct {
	client *headchef_operations.Client
}

func InitClient() *Client {
	return NewClient(api.GetServiceURL(api.ServiceHeadChef))
}

func NewClient(apiURL *url.URL) *Client {
	transportRuntime := httptransport.New(apiURL.Host, apiURL.Path, []string{apiURL.Scheme})
	transportRuntime.Transport = api.NewUserAgentTripper()

	//transportRuntime.SetDebug(true)

	return &Client{
		client: headchef_client.New(transportRuntime, strfmt.Default).HeadchefOperations,
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

func (r *Client) reqBuild(buildReq *headchef_models.V1BuildRequest, buildStatus *BuildStatus) {
	startParams := headchef_operations.StartBuildV1Params{
		Context:      context.Background(),
		BuildRequest: buildReq,
	}
	created, accepted, err := r.client.StartBuildV1(&startParams)

	switch {
	case err != nil:
		msg := err.Error()
		if startErr, ok := err.(*headchef_operations.StartBuildV1Default); ok {
			msg = *startErr.Payload.Message
		}
		buildStatus.RunFail <- FailBuildReqErrorResp.New(msg)
	case accepted != nil:
		buildStatus.Started <- struct{}{}
	case created != nil:
		if created.Payload.Type == nil {
			msg := "created response cannot be handled: nil type"
			buildStatus.RunFail <- FailBuildCreatedNilType.New(msg)
			break
		}
		payloadType := *created.Payload.Type

		switch payloadType {
		case headchef_models.BuildStatusResponseTypeBuildCompleted:
			buildStatus.Completed <- created.Payload
		case headchef_models.BuildStatusResponseTypeBuildFailed:
			buildStatus.Failed <- created.Payload.Message
		case headchef_models.BuildStatusResponseTypeBuildStarted:
			buildStatus.Started <- struct{}{}
		default:
			msg := fmt.Sprintf(
				"created response cannot be handled: unknown type %q",
				payloadType,
			)
			buildStatus.RunFail <- FailBuildCreatedUnknownType.New(msg)
		}
	default:
		buildStatus.RunFail <- FailBuildReqNoResp.New("no response")
	}
}
