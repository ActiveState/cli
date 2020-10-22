package packages

import (
	"github.com/ActiveState/cli/internal/headless"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// AddRunParams tracks the info required for running Add.
type AddRunParams struct {
	Name string
}

// Add manages the adding execution context.
type Add struct {
	out  output.Outputer
	proj *project.Project
	prompt.Prompter
	auth *authentication.Auth
}

// NewAdd prepares an addition execution context for use.
func NewAdd(prime primeable) *Add {
	return &Add{
		prime.Output(),
		prime.Project(),
		prime.Prompt(),
		prime.Auth(),
	}
}

// Run executes the add behavior.
func (a *Add) Run(params AddRunParams) error {
	err := a.run(params)
	headless.Notify(a.out, a.proj, err, "packages")
	return err
}

func (a *Add) run(params AddRunParams) error {
	logging.Debug("ExecuteAddUpdate")

	language, fail := model.LanguageForCommit(a.proj.CommitUUID())
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_fetch_languages")
	}

	name, version := splitNameAndVersion(params.Name)

	operation := model.OperationAdded
	hasPkg, err := model.HasPackage(a.proj.CommitUUID(), name)
	if err != nil {
		return locale.WrapError(
			err, "err_checking_package_exists",
			"Cannot verify if package is already in use.",
		)
	}
	if hasPkg {
		operation = model.OperationUpdated
	}

	return executePackageOperation(a.proj, a.out, a.auth, a.Prompter, language, name, version, operation)
}
