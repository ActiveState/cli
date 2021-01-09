package packages

import (
	"github.com/ActiveState/cli/internal/keypairs"
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
	cfg  keypairs.Configurable
	out  output.Outputer
	proj *project.Project
	prompt.Prompter
	auth *authentication.Auth
}

// NewInstall prepares an installation execution context for use.
func NewInstall(prime primeable) *Install {
	return &Install{
		prime.Config(),
		prime.Output(),
		prime.Project(),
		prime.Prompt(),
		prime.Auth(),
	}
}

// Run executes the install behavior.
func (a *Install) Run(params InstallRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteInstall")
	if a.proj == nil {
		return locale.NewInputError("err_no_project")
	}

	language, err := model.LanguageForCommit(a.proj.CommitUUID())
	if err != nil {
		return locale.WrapError(err, "err_fetch_languages")
	}

	ns := model.NewNamespacePkgOrBundle(language, nstype)
	name, version := splitNameAndVersion(params.Name)
	return executePackageOperation(a.proj, a.cfg, a.out, a.auth, a.Prompter, name, version, model.OperationAdded, ns)
}
