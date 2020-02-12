package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// RunAddParams tracks the info required for running Add.
type RunAddParams struct {
	Params
}

// Add manages the adding execution context.
type Add struct {
	getProject ProjectProviderFunc
}

// NewAdd prepares an add execution context for use.
func NewAdd(getProjFn ProjectProviderFunc) *Add {
	return &Add{
		getProject: getProjFn,
	}
}

// Run executes the add behavior.
func (a *Add) Run(ps RunAddParams) error {
	logging.Debug("Execute platforms add")

	params, err := prepareParams(ps.Params)
	if err != nil {
		return nil
	}

	proj, fail := a.getProject()
	if fail != nil {
		return fail
	}

	return model.CommitPlatform(
		proj.Owner(), proj.Name(),
		model.OperationAdded,
		params.Name, params.Version, params.BitWidth,
	)
}
