package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RunRemoveParams tracks the info required for running Remove.
type RunRemoveParams struct {
	Params
}

// Remove manages the removeing execution context.
type Remove struct{}

// NewRemove prepares a remove execution context for use.
func NewRemove() *Remove {
	return &Remove{}
}

// Run executes the remove behavior.
func (r *Remove) Run(params RunRemoveParams) error {
	logging.Debug("Execute platforms remove")

	return remove(params.Params)
}

func remove(ps Params) error {
	params, err := prepareParams(ps)
	if err != nil {
		return nil
	}

	proj := project.Get()

	return model.CommitPlatform(
		proj.Owner(), proj.Name(),
		model.OperationRemoved,
		params.Name, params.Version, params.BitWidth,
	)
}
