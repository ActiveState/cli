package request

import "github.com/ActiveState/cli/internal/gqlclient"

type DeprecationRequest struct {
	gqlclient.RequestBase
}

func NewDeprecationRequest() *DeprecationRequest {
	return &DeprecationRequest{}
}

func (d *DeprecationRequest) Query() string {
	return `query {
		checkDeprecation {
			version
			date
			dateReached
			reason
		}
	}`
}

func (d *DeprecationRequest) Vars() map[string]interface{} {
	return nil
}
