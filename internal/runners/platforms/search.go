package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// availableProvider describes the behavior needed to obtain available platforms.
type availableProvider interface {
	AvailablePlatforms() ([]*model.Platform, error)
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

	fetch := &fetchAvailable{}

	return newSearchResult(fetch)
}

// SearchResult represents the output data of a search.
type SearchResult struct {
	Platforms []*Platform `json:"platforms"`
}

func newSearchResult(p availableProvider) (*SearchResult, error) {
	platforms, err := p.AvailablePlatforms()
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
func (f *fetchAvailable) AvailablePlatforms() ([]*model.Platform, error) {
	platforms, fail := model.FetchPlatforms()
	if fail != nil {
		return nil, fail
	}

	return platforms, nil
}
