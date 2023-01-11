package packages

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type PackageVersion struct {
	captain.NameVersion
}

func (pv *PackageVersion) Set(arg string) error {
	err := pv.NameVersion.Set(arg)
	if err != nil {
		return locale.WrapInputError(err, "err_package_format", "The package and version provided is not formatting correctly, must be in the form of <package>@<version>")
	}
	return nil
}

// InstallRunParams tracks the info required for running Install.
type InstallRunParams struct {
	Package PackageVersion
}

// Install manages the installing execution context.
type Install struct {
	output    output.Outputer
	prompt    prompt.Prompter
	project   *project.Project
	auth      *authentication.Auth
	config    *config.Instance
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
}

// NewInstall prepares an installation execution context for use.
func NewInstall(prime primeable) *Install {
	return &Install{
		output:    prime.Output(),
		prompt:    prime.Prompt(),
		project:   prime.Project(),
		auth:      prime.Auth(),
		config:    prime.Config(),
		analytics: prime.Analytics(),
		svcModel:  prime.SvcModel(),
	}
}

// Run executes the install behavior.
func (a *Install) Run(params InstallRunParams, nsType model.NamespaceType) error {
	logging.Debug("ExecuteInstall")
	return requirements.ExecuteRequirementOperation(&requirements.RequirementOperationParams{
		Output:             a.output,
		Prompt:             a.prompt,
		Project:            a.project,
		Auth:               a.auth,
		Config:             a.config,
		Analytics:          a.analytics,
		SvcModel:           a.svcModel,
		RequirementName:    params.Package.Name(),
		RequirementVersion: params.Package.Version(),
		Operation:          model.OperationAdded,
		NsType:             model.NamespaceLanguage,
	})
}
