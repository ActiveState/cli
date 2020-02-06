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

func newSearchResult(fetcher Fetcher) (*SearchResult, error) {
	platforms, fail := fetcher.FetchPlatforms()
	if fail != nil {
		return nil, fail
	}

	var result SearchResult
	for _, platform := range platforms {
		var p Platform
		if platform.DisplayName != nil {
			p.Name = *platform.DisplayName
		}
		result.Platforms = append(result.Platforms, &p)
	}

	return &result, nil
}
