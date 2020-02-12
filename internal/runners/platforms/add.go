package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RunAddParams tracks the info required for running Add.
type RunAddParams struct {
	Params
}

// Add manages the adding execution context.
type Add struct{}

// NewAdd prepares an add execution context for use.
func NewAdd() *Add {
	return &Add{}
}

// Run executes the add behavior.
func (a *Add) Run(params RunAddParams) error {
	logging.Debug("Execute platforms add")

	return add(params.Params)
}

func add(ps Params) error {
	params, err := prepareParams(ps)
	if err != nil {
		return nil
	}

	proj := project.Get()

	return model.CommitPlatform(
		proj.Owner(), proj.Name(),
		model.OperationAdded,
		params.Name, params.Version, params.BitWidth,
	)
}
