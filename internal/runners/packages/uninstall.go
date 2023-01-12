package packages

import (
	"github.com/ActiveState/cli/internal/analytics"
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

// UninstallRunParams tracks the info required for running Uninstall.
type UninstallRunParams struct {
	Name string
}

// Uninstall manages the uninstalling execution context.
type Uninstall struct {
	output    output.Outputer
	prompt    prompt.Prompter
	project   *project.Project
	auth      *authentication.Auth
	config    *config.Instance
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
}

// NewUninstall prepares an uninstallation execution context for use.
func NewUninstall(prime primeable) *Uninstall {
	return &Uninstall{
		output:    prime.Output(),
		prompt:    prime.Prompt(),
		project:   prime.Project(),
		auth:      prime.Auth(),
		config:    prime.Config(),
		analytics: prime.Analytics(),
		svcModel:  prime.SvcModel(),
	}
}

// Run executes the uninstall behavior.
func (r *Uninstall) Run(params UninstallRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteUninstall")
	if r.project == nil {
		return locale.NewInputError("err_no_project")
	}

	return requirements.ExecuteRequirementOperation(&requirements.RequirementOperationParams{
		Output:          r.output,
		Prompt:          r.prompt,
		Project:         r.project,
		Auth:            r.auth,
		Config:          r.config,
		Analytics:       r.analytics,
		SvcModel:        r.svcModel,
		RequirementName: params.Name,
		Operation:       model.OperationRemoved,
		NsType:          model.NamespacePackage,
	})
}
