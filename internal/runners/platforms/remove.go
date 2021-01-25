package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RemoveRunParams tracks the info required for running Remove.
type RemoveRunParams struct {
	Params
	Project *project.Project
}

// Remove manages the removeing execution context.
type Remove struct{}

// NewRemove prepares a remove execution context for use.
func NewRemove() *Remove {
	return &Remove{}
}

// Run executes the remove behavior.
func (r *Remove) Run(ps RemoveRunParams) error {
	logging.Debug("Execute platforms remove")

	params, err := prepareParams(ps.Params)
	if err != nil {
		return nil
	}

	return model.CommitPlatform(
		ps.Project.Owner(), ps.Project.Name(),
		model.OperationRemoved,
		params.name, params.version, params.BitWidth,
	)
}
