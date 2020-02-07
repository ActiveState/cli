package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// availableFetcher describes the behavior needed to obtain platforms.
type availableFetcher interface {
	FetchAvailablePlatforms() ([]*model.Platform, error)
}

// Search manages the searching execution context.
type Search struct{}

// NewSearch prepares a search execution context for use.
func NewSearch() *Search {
	return &Search{}
}

// Run executes the search behavior.
func (s *Search) Run() (*SearchResult, error) {
	logging.Debug("Execute platforms search")

	fetcher := &fetchAvailable{}

	return newSearchResult(fetcher)
}

// SearchResult represents the output data of a search.
type SearchResult struct {
	Platforms []*Platform `json:"platforms"`
}

func newSearchResult(fetcher availableFetcher) (*SearchResult, error) {
	platforms, err := fetcher.FetchAvailablePlatforms()
	if err != nil {
		return nil, err
	}

	result := SearchResult{
		Platforms: makePlatformsFromModelPlatforms(platforms),
	}

	return &result, nil
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
