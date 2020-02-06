package platforms

import "github.com/ActiveState/cli/pkg/platform/model"

// Fetcher describes the behavior needed to obtain platforms.
type Fetcher interface {
	FetchPlatforms() ([]*model.Platform, error)
}

// Platform represents the output data of a platform.
type Platform struct {
	Name string `json:"name"`
}

type fetch struct{}

// FetchPlatforms implements the Fetcher interface.
func (f *fetch) FetchPlatforms() ([]*model.Platform, error) {
	platforms, fail := model.FetchPlatforms()
	if fail != nil {
		return nil, fail
	}

	return platforms, nil
}
