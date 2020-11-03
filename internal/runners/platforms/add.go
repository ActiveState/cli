package platforms

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// AddRunParams tracks the info required for running Add.
type AddRunParams struct {
	Params
	Project *project.Project
}

// Add manages the adding execution context.
type Add struct {
	output.Outputer
}

type primeable interface {
	primer.Outputer
}

// NewAdd prepares an add execution context for use.
func NewAdd(prime primeable) *Add {
	return &Add{prime.Output()}
}

// Run executes the add behavior.
func (a *Add) Run(ps AddRunParams) error {
	logging.Debug("Execute platforms add")

	params, err := prepareParams(ps.Params)
	if err != nil {
		return err
	}

	modifiable, err := model.IsProjectModifiable(ps.Project.Owner(), ps.Project.Name())
	if err != nil {
		return locale.WrapError(err, "err_modifiable", "Could not determine if project is modifiable")
	}
	if !modifiable {
		return locale.NewError(
			"err_not_modifiable",
			"You do not have permission to modify the project at [NOTICE]{{.V0}}/{{.V1}}[/RESET]. You will either need to be invited to this project or you can fork it by running `[ACTIONABLE]state fork {{.V0}}/{{.V1}}[/RESET].`",
			ps.Project.Owner(), ps.Project.Name(),
		)
	}

	fail := model.CommitPlatform(
		ps.Project.Owner(), ps.Project.Name(),
		model.OperationAdded,
		params.Name, params.Version, params.BitWidth,
	)
	if fail != nil {
		return fail
	}

	a.Outputer.Notice(locale.Tr("platform_added", params.Name, params.Version))

	return nil
}
