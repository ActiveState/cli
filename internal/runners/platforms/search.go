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

	res, err := newSearchOutput()
	if err != nil {
		return err
	}

	s.out.Print(res)
	return nil
}

type searchOutput struct {
	Platforms []*Platform `json:"platforms"`
}

func (o *searchOutput) MarshalStructured(format output.Format) interface{} {
	return o
}

func newSearchOutput() (*searchOutput, error) {
	platforms, err := model.FetchPlatforms()
	if err != nil {
		return nil, err
	}

	result := searchOutput{
		Platforms: makePlatformsFromModelPlatforms(platforms),
	}

	return &result, nil
}
