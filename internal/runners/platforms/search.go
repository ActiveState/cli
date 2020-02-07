package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
)

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
