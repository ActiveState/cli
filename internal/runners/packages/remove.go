package packages

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RemoveRunParams tracks the info required for running Remove.
type RemoveRunParams struct {
	Name string
}

// Remove manages the removing execution context.
type Remove struct{}

// NewRemove prepares a removal execution context for use.
func NewRemove() *Remove {
	return &Remove{}
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
	print.Line(locale.Tr("package_removed", params.Name))

	// Remind user to update their activestate.yaml
	print.Warning(locale.T("package_update_config_file"))

	return nil
}
