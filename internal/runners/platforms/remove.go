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

	return model.CommitPlatform(
		ps.Project.Owner(), ps.Project.Name(),
		model.OperationRemoved,
		params.Name, params.Version, params.BitWidth,
	)
}
