package packages

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// InstallRunParams tracks the info required for running Install.
type InstallRunParams struct {
	Name string
}

// Install manages the installing execution context.
type Install struct {
	out  output.Outputer
	proj *project.Project
	prompt.Prompter
	auth *authentication.Auth
}

// NewInstall prepares an installation execution context for use.
func NewInstall(prime primeable) *Install {
	return &Install{
		prime.Output(),
		prime.Project(),
		prime.Prompt(),
		prime.Auth(),
	}
}

// Run executes the install behavior.
func (a *Install) Run(params InstallRunParams) error {
	logging.Debug("ExecuteInstall")
	if a.proj == nil {
		return locale.NewError("package_operation_no_project")
	}

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
