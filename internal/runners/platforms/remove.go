package platforms

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RemoveRunParams tracks the info required for running Remove.
type RemoveRunParams struct {
	Params
}

// Remove manages the removeing execution context.
type Remove struct {
	output    output.Outputer
	prompt    prompt.Prompter
	project   *project.Project
	auth      *authentication.Auth
	config    *config.Instance
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
}

// NewRemove prepares a remove execution context for use.
func NewRemove(prime primeable) *Remove {
	return &Remove{
		output:    prime.Output(),
		prompt:    prime.Prompt(),
		project:   prime.Project(),
		auth:      prime.Auth(),
		config:    prime.Config(),
		analytics: prime.Analytics(),
		svcModel:  prime.SvcModel(),
	}
}

// Run executes the remove behavior.
func (r *Remove) Run(ps RemoveRunParams) error {
	logging.Debug("Execute platforms remove")

	if r.project == nil {
		return locale.NewInputError("err_no_project")
	}

	params, err := prepareParams(ps.Params)
	if err != nil {
		return errs.Wrap(err, "Could not prepare parameters.")
	}

	if err := requirements.ExecuteRequirementOperation(&requirements.RequirementOperationParams{
		Output:              r.output,
		Prompt:              r.prompt,
		Project:             r.project,
		Auth:                r.auth,
		Config:              r.config,
		Analytics:           r.analytics,
		SvcModel:            r.svcModel,
		RequirementName:     params.name,
		RequirementVersion:  params.version,
		RequirementBitWidth: params.BitWidth,
		Operation:           model.OperationAdded,
		NsType:              model.NamespaceLanguage,
	}); err != nil {
		return locale.WrapError(err, "err_remove_platform", "Could not remove platform.")
	}

	r.output.Notice(locale.Tr("platform_removed", params.name, params.version))

	return nil
}
