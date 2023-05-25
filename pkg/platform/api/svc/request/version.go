package request

import "github.com/ActiveState/cli/internal/gqlclient"

type VersionRequest struct {
	gqlclient.RequestBase
}

func NewVersionRequest() *VersionRequest {
	return &VersionRequest{}
}

func (v *VersionRequest) Query() string {
	return `query {
        version {
            state {
                license,
                version,
                branch,
                revision,
                date,
            }
        }
    }`
}

func (v *VersionRequest) Vars() (map[string]interface{}, error) {
	return nil, nil
}
