package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
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
	platforms, fail := model.FetchPlatforms()
	if fail != nil {
		return nil, fail
	}

	result := SearchResult{
		Platforms: MakePlatformsFromModelPlatforms(platforms),
	}

	return &result, nil
}
