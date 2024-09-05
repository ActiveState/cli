package platforms

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime/requirements"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/model"
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
	return &Remove{
		prime: prime,
	}
}

// Run executes the remove behavior.
func (r *Remove) Run(ps RemoveRunParams) error {
	logging.Debug("Execute platforms remove")

	if r.prime.Project() == nil {
		return rationalize.ErrNoProject
	}

	params, err := prepareParams(ps.Params)
	if err != nil {
		return errs.Wrap(err, "Could not prepare parameters.")
	}

	if err := requirements.NewRequirementOperation(r.prime).ExecuteRequirementOperation(
		&requirements.Requirement{
			Name:          params.name,
			Version:       params.version,
			Operation:     types.OperationRemoved,
			BitWidth:      params.BitWidth,
			NamespaceType: &model.NamespacePlatform,
		},
	); err != nil {
		return locale.WrapError(err, "err_remove_platform", "Could not remove platform.")
	}

	r.prime.Output().Notice(locale.Tr("platform_removed", params.name, params.version))

	return nil
}
