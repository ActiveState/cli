package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// RunRemoveParams tracks the info required for running Remove.
type RunRemoveParams struct {
	Params
}

// Remove manages the removeing execution context.
type Remove struct {
	getProject ProjectProviderFunc
}

// NewRemove prepares a remove execution context for use.
func NewRemove(getProjFn ProjectProviderFunc) *Remove {
	return &Remove{
		getProject: getProjFn,
	}
}

// Run executes the remove behavior.
func (r *Remove) Run(ps RunRemoveParams) error {
	logging.Debug("Execute platforms remove")

	params, err := prepareParams(ps.Params)
	if err != nil {
		return nil
	}

	proj, fail := r.getProject()
	if fail != nil {
		return fail
	}
	return model.CommitPlatform(
		proj.Owner(), proj.Name(),
		model.OperationRemoved,
		params.Name, params.Version, params.BitWidth,
	)
}
