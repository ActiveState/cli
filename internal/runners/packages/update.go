package packages

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// UpdateRunParams tracks the info required for running Update.
type UpdateRunParams struct {
	Name string
}

// Update manages the updating execution context.
type Update struct {
	out output.Outputer
	prompt.Prompter
}

// NewUpdate prepares an update execution context for use.
func NewUpdate(prime primeable) *Update {
	return &Update{
		prime.Output(),
		prime.Prompt(),
	}
}

// Run executes the update behavior.
func (u *Update) Run(params UpdateRunParams) error {
	logging.Debug("ExecuteUpdate")

	pj := project.Get()
	language, fail := model.DefaultLanguageForProject(pj.Owner(), pj.Name())
	if fail != nil {
		return fail.WithDescription("err_fetch_languages")
	}

	name, version := splitNameAndVersion(params.Name)
	if version == "" {
		ingredientVersion, err := model.IngredientWithLatestVersion(language, name)
		if err != nil {
			return locale.WrapError(err, "package_ingredient_err", "Failed to resolve an ingredient named {{.V0}}.", name)
		}
		version = *ingredientVersion.Version.Version
	}

	return executeAddUpdate(u.out, u.Prompter, language, name, version, model.OperationUpdated)
}
