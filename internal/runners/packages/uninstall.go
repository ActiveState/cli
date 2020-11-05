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

// UninstallRunParams tracks the info required for running Uninstall.
type UninstallRunParams struct {
	Name string
}

// Uninstall manages the uninstalling execution context.
type Uninstall struct {
	out  output.Outputer
	proj *project.Project
	prompt.Prompter
	auth *authentication.Auth
}

// NewUninstall prepares an uninstallation execution context for use.
func NewUninstall(prime primeable) *Uninstall {
	return &Uninstall{
		prime.Output(),
		prime.Project(),
		prime.Prompt(),
		prime.Auth(),
	}
}

// Run executes the uninstall behavior.
func (r *Uninstall) Run(params UninstallRunParams) error {
	logging.Debug("ExecuteUninstall")
	if r.proj == nil {
		return locale.NewInputError("err_no_project")
	}

	// Commit the package
	language, fail := model.LanguageForCommit(r.proj.CommitUUID())
	if fail != nil {
		return locale.WrapError(fail, "err_fetch_languages")
	}

	return executePackageOperation(r.proj, r.out, r.auth, r.Prompter, language, params.Name, "", model.OperationRemoved)
}
