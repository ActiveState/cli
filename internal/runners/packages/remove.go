package packages

import (
	"github.com/ActiveState/cli/internal/headless"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
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
	out  output.Outputer
	proj *project.Project
	prompt.Prompter
}

// NewRemove prepares a removal execution context for use.
func NewRemove(prime primeable) *Remove {
	return &Remove{
		prime.Output(),
		prime.Project(),
		prime.Prompt(),
	}
}

// Run executes the remove behavior.
func (r *Remove) Run(params RemoveRunParams) error {
	err := r.run(params)
	headless.Notify(r.out, r.proj, err, "packages")
	return err
}

func (r *Remove) run(params RemoveRunParams) error {
	fail := auth.RequireAuthentication(locale.T("auth_required_activate"), r.out, r.Prompter)
	if fail != nil {
		return fail.WithDescription("err_activate_auth_required").ToError()
	}

	// Commit the package
	pj := project.Get()
	language, fail := model.DefaultLanguageNameForProject(pj.Owner(), pj.Name())
	if fail != nil {
		return locale.WrapError(fail, "err_fetch_languages")
	}

	return executePackageOperation(r.out, r.Prompter, language, params.Name, "", model.OperationRemoved)
}
