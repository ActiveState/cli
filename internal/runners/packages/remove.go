package packages

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// RemoveRunParams tracks the info required for running Remove.
type RemoveRunParams struct {
	Name string
}

// Remove manages the removing execution context.
type Remove struct {
	out output.Outputer
	prompt.Prompter
}

// NewRemove prepares a removal execution context for use.
func NewRemove(prime primeable) *Remove {
	return &Remove{
		prime.Output(),
		prime.Prompt(),
	}
}

// Run executes the remove behavior.
func (r *Remove) Run(params RemoveRunParams) error {
	logging.Debug("ExecuteRemove")

	return executePackageOperation(r.out, r.Prompter, "", params.Name, "", model.OperationRemoved)
}
