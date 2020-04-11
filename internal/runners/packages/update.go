package packages

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// UpdateRunParams tracks the info required for running Update.
type UpdateRunParams struct {
	Name string
}

// Update manages the updating execution context.
type Update struct{}

// NewUpdate prepares an update execution context for use.
func NewUpdate() *Update {
	return &Update{}
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
		ingredientVersion, fail := model.IngredientWithLatestVersion(language, name)
		if fail != nil {
			return fail.WithDescription("package_ingredient_not_found")
		}
		version = *ingredientVersion.Version.Version
	}

	return executeAddUpdate(language, name, version, model.OperationUpdated)
}
