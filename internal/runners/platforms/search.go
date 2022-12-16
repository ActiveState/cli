package platforms

import (
	"github.com/ActiveState/cli/internal-as/logging"
	"github.com/ActiveState/cli/internal-as/output"
	"github.com/ActiveState/cli/internal-as/primer"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// Search manages the searching execution context.
type Search struct {
	out output.Outputer
}

// NewSearch prepares a search execution context for use.
func NewSearch(prime primer.Outputer) *Search {
	return &Search{
		out: prime.Output(),
	}
}

// Run executes the search behavior.
func (s *Search) Run() error {
	logging.Debug("Execute platforms search")

	res, err := newSearchResult()
	if err != nil {
		return err
	}

	s.out.Print(res)
	return nil
}

// SearchResult represents the output data of a search.
type SearchResult struct {
	Platforms []*Platform `json:"platforms"`
}

func newSearchResult() (*SearchResult, error) {
	platforms, err := model.FetchPlatforms()
	if err != nil {
		return nil, err
	}

	result := SearchResult{
		Platforms: makePlatformsFromModelPlatforms(platforms),
	}

	return &result, nil
}
