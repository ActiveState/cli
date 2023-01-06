package platforms

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RemoveRunParams tracks the info required for running Remove.
type RemoveRunParams struct {
	Params
}

// Remove manages the removeing execution context.
type Remove struct {
	prime primeable
}

// NewRemove prepares a remove execution context for use.
func NewRemove(prime primeable) *Remove {
	return &Remove{prime}
}

// Run executes the remove behavior.
func (r *Remove) Run(ps RemoveRunParams) error {
	logging.Debug("Execute platforms remove")

	if r.prime.Project() == nil {
		return locale.NewInputError("err_no_project")
	}

	params, err := prepareParams(ps.Params)
	if err != nil {
		return nil
	}

	err = requirements.ExecuteRequirementOperation(r.prime, params.name, params.version, params.BitWidth, model.OperationRemoved, model.NamespacePlatform)
	if err != nil {
		return locale.WrapError(err, "err_remove_platform", "Could not remove platform.")
	}

	r.prime.Output().Notice(locale.Tr("platform_removed", params.name, params.version))

	return nil
}
