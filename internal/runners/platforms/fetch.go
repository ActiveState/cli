package platforms

import (
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/go-openapi/strfmt"
)

// fetcher describes the behavior needed to obtain platforms.
type availableFetcher interface {
	FetchAvailablePlatforms() ([]*model.Platform, error)
}

type fetchAvailable struct{}

// FetchAvailablePlatforms implements the availableFetcher interface.
func (f *fetchAvailable) FetchAvailablePlatforms() ([]*model.Platform, error) {
	platforms, fail := model.FetchPlatforms()
	if fail != nil {
		return nil, fail
	}

	return platforms, nil
}

type committedFetcher interface {
	FetchCommittedPlatforms(strfmt.UUID) ([]*model.Platform, error)
}

type fetchCommitted struct{}

func (f *fetchCommitted) FetchCommittedPlatforms(commitID strfmt.UUID) ([]*model.Platform, error) {
	platforms, fail := model.FetchPlatformsForCommit(commitID)
	if fail != nil {
		return nil, fail
	}

	return platforms, nil
}
