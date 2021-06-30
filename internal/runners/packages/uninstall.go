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
	cfg  configurable
	out  output.Outputer
	proj *project.Project
	prompt.Prompter
	auth *authentication.Auth
}

// NewUninstall prepares an uninstallation execution context for use.
func NewUninstall(prime primeable) *Uninstall {
	return &Uninstall{
		prime.Config(),
		prime.Output(),
		prime.Project(),
		prime.Prompt(),
		prime.Auth(),
	}
}

// Run executes the uninstall behavior.
func (r *Uninstall) Run(params UninstallRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteUninstall")
	if r.proj == nil {
		return locale.NewInputError("err_no_project")
	}

	return executePackageOperation(r.proj, r.cfg, r.out, r.auth, r.Prompter, params.Name, "", model.OperationRemoved, nstype)
}
