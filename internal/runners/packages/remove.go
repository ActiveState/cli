package packages

import (
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
	out output.Outputer
	prompt.Prompter
}

// NewRemove prepares a removal execution context for use.
func NewRemove(prime primeable) *Remove {
	return &Remove{
		prime.Output(),
		prime.Prompt(),
	}
}

// Run executes the remove behavior.
func (r *Remove) Run(params RemoveRunParams) error {
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

	ingredient, err := model.IngredientWithLatestVersion(language, params.Name)
	if err != nil {
		return locale.WrapError(err, "err_remove_get_package", "Could not find package to remove")
	}

	fail = model.CommitPackage(pj.Owner(), pj.Name(), model.OperationRemoved, params.Name, ingredient.Namespace, "")
	if fail != nil {
		return fail.WithDescription("err_package_removed").ToError()
	}

	// Print the result
	r.out.Print(locale.Tr("package_removed", params.Name))

	// Remind user to update their activestate.yaml
	r.out.Notice(locale.T("package_update_config_file"))

	return nil
}
