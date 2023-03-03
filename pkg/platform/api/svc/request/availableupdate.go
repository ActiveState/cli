package request

import "github.com/ActiveState/cli/internal/gqlclient"

type AvailableUpdate struct {
	gqlclient.RequestBase
}

func NewAvailableUpdate() *AvailableUpdate {
	return &AvailableUpdate{}
}

func (u *AvailableUpdate) Query() string {
	return `query() {
		availableUpdate() {
			version
			channel
			path
			platform
			sha256
		}
	}`
}

func (u *AvailableUpdate) Vars() map[string]interface{} {
	return nil
}
