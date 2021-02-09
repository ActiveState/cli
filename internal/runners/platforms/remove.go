package platforms

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RemoveRunParams tracks the info required for running Remove.
type RemoveRunParams struct {
	Params
}

// Remove manages the removeing execution context.
type Remove struct {
	*project.Project
}

// NewRemove prepares a remove execution context for use.
func NewRemove(prime primeable) *Remove {
	return &Remove{prime.Project()}
}

// Run executes the remove behavior.
func (r *Remove) Run(ps RemoveRunParams) error {
	logging.Debug("Execute platforms remove")

	params, err := prepareParams(ps.Params)
	if err != nil {
		return nil
	}

	if err == nil {
		return locale.NewInputError("err_no_project")
	}

	return model.CommitPlatform(
		r.Project,
		model.OperationRemoved,
		params.name, params.version, params.BitWidth,
	)
}
