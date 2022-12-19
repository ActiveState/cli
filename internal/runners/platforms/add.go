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
}

// Add manages the adding execution context.
type Add struct {
	out     output.Outputer
	project *project.Project
}

type primeable interface {
	primer.Outputer
	primer.Projecter
}

// NewAdd prepares an add execution context for use.
func NewAdd(prime primeable) *Add {
	return &Add{prime.Output(), prime.Project()}
}

// Run executes the add behavior.
func (a *Add) Run(ps AddRunParams) error {
	logging.Debug("Execute platforms add")

	params, err := prepareParams(ps.Params)
	if err != nil {
		return err
	}

	if a.project == nil {
		return locale.NewInputError("err_no_project")
	}

	commit, err := model.CommitPlatform(
		a.project.CommitUUID(),
		model.OperationAdded,
		params.name, params.version, params.BitWidth,
	)
	if err != nil {
		return locale.WrapError(err, "err_add_platform", "Could not add platform.")
	}

	if err := a.project.SetCommit(commit.CommitID.String()); err != nil {
		return locale.WrapError(err, "err_package_update_pjfile")
	}

	a.out.Notice(locale.Tr("platform_added", params.name, params.version))

	return nil
}
