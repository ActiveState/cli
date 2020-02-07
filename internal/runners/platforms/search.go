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

	fetcher := &fetch{}

	return newSearchResult(fetcher)
}

// SearchResult represents the output data of a search.
type SearchResult struct {
	Platforms []*Platform `json:"platforms"`
}

func newSearchResult(f fetcher) (*SearchResult, error) {
	platforms, err := f.FetchPlatforms()
	if err != nil {
		return nil, err
	}

	result := SearchResult{
		Platforms: makePlatformsFromModelPlatforms(platforms),
	}

	return &result, nil
}
