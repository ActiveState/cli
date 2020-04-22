package platforms

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// AddRunParams tracks the info required for running Add.
type AddRunParams struct {
	Params
	Project *project.Project
}

// Add manages the adding execution context.
type Add struct{}

// NewAdd prepares an add execution context for use.
func NewAdd() *Add {
	return &Add{}
}

// Run executes the add behavior.
func (a *Add) Run(ps AddRunParams) error {
	logging.Debug("Execute platforms add")

	params, err := prepareParams(ps.Params)
	if err != nil {
		return err
	}

	fail := model.CommitPlatform(
		ps.Project.Owner(), ps.Project.Name(),
		model.OperationAdded,
		params.Name, params.Version, params.BitWidth,
	)
	if fail != nil {
		return fail
	}

	print.Line(locale.Tr("platform_added", params.Name, params.Version))

	return nil
}
