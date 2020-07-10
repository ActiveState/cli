package packages

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RemoveRunParams tracks the info required for running Remove.
type RemoveRunParams struct {
	Name string
}

// Remove manages the removing execution context.
type Remove struct {
	out output.Outputer
}

// NewRemove prepares a removal execution context for use.
func NewRemove(prime primer.Outputer) *Remove {
	return &Remove{
		out: prime.Output(),
	}
}

// Run executes the remove behavior.
func (r *Remove) Run(params RemoveRunParams) error {
	fail := auth.RequireAuthentication(locale.T("auth_required_activate"))
	if fail != nil {
		return fail.WithDescription("err_activate_auth_required")
	}

	// Commit the package
	pj := project.Get()
	fail = model.CommitPackage(pj.Owner(), pj.Name(), model.OperationRemoved, params.Name, "")
	if fail != nil {
		return fail.WithDescription("err_package_removed")
	}

	// Print the result
	r.out.Print(locale.Tr("package_removed", params.Name))

	// Remind user to update their activestate.yaml
	r.out.Notice(locale.T("package_update_config_file"))

	return nil
}
