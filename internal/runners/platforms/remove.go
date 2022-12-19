package platforms

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RemoveRunParams tracks the info required for running Remove.
type RemoveRunParams struct {
	Params
}

// Remove manages the removeing execution context.
type Remove struct {
	project *project.Project
	out     output.Outputer
}

// NewRemove prepares a remove execution context for use.
func NewRemove(prime primeable) *Remove {
	return &Remove{prime.Project(), prime.Output()}
}

// Run executes the remove behavior.
func (r *Remove) Run(ps RemoveRunParams) error {
	logging.Debug("Execute platforms remove")

	if r.project == nil {
		return locale.NewInputError("err_no_project")
	}

	params, err := prepareParams(ps.Params)
	if err != nil {
		return nil
	}

	commit, err := model.CommitPlatform(
		r.project.CommitUUID(),
		model.OperationRemoved,
		params.name, params.version, params.BitWidth,
	)
	if err != nil {
		return locale.WrapError(err, "err_remove_platform", "Could not remove platform.")
	}

	if err := r.project.SetCommit(commit.CommitID.String()); err != nil {
		return locale.WrapError(err, "err_package_update_pjfile")
	}

	r.out.Notice(locale.Tr("platform_removed", params.name, params.version))

	return nil
}
