package platforms

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// Search manages the searching execution context.
type Search struct {
	out  output.Outputer
	auth *authentication.Auth
}

// NewSearch prepares a search execution context for use.
func NewSearch(prime primeable) *Search {
	return &Search{
		out:  prime.Output(),
		auth: prime.Auth(),
	}
}

// Run executes the search behavior.
func (s *Search) Run() error {
	logging.Debug("Execute platforms search")

	modelPlatforms, err := model.FetchPlatforms(s.auth)
	if err != nil {
		return errs.Wrap(err, "Unable to fetch platforms")
	}

	platforms := makePlatformsFromModelPlatforms(modelPlatforms)
	s.out.Print(output.Prepare(platforms, platforms))
	return nil
}
