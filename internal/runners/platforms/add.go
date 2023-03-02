package platforms

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	bgModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// AddRunParams tracks the info required for running Add.
type AddRunParams struct {
	Params
}

// Add manages the adding execution context.
type Add struct {
	prime primeable
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Projecter
	primer.Auther
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
}

// NewAdd prepares an add execution context for use.
func NewAdd(prime primeable) *Add {
	return &Add{
		prime: prime,
	}
}

// Run executes the add behavior.
func (a *Add) Run(ps AddRunParams) error {
	logging.Debug("Execute platforms add")

	params, err := prepareParams(ps.Params)
	if err != nil {
		return err
	}

	if a.prime.Project() == nil {
		return locale.NewInputError("err_no_project")
	}

	if err := requirements.NewRequirementOperation(a.prime).ExecuteRequirementOperation(
		params.name,
		params.version,
		"",
		params.BitWidth,
		bgModel.OperationAdd,
		model.NamespacePlatform,
	); err != nil {
		return locale.WrapError(err, "err_add_platform", "Could not add platform.")
	}

	a.prime.Output().Notice(locale.Tr("platform_added", params.name, params.version))

	return nil
}
