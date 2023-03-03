package request

import "github.com/ActiveState/cli/internal/gqlclient"

type LocalProjectsRequest struct {
	gqlclient.RequestBase
}

func NewLocalProjectsRequest() *LocalProjectsRequest {
	return &LocalProjectsRequest{}
}

func (l *LocalProjectsRequest) Query() string {
	return `query {
		projects {
			namespace
			locations
		}
	}`
}

func (l *LocalProjectsRequest) Vars() map[string]interface{} {
	return nil
}
