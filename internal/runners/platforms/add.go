package platforms

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// AddRunParams tracks the info required for running Add.
type AddRunParams struct {
	Params
}

// Add manages the adding execution context.
type Add struct {
	output    output.Outputer
	prompt    prompt.Prompter
	project   *project.Project
	auth      *authentication.Auth
	config    *config.Instance
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Projecter
	primer.Auther
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
}

// NewAdd prepares an add execution context for use.
func NewAdd(prime primeable) *Add {
	return &Add{
		output:    prime.Output(),
		prompt:    prime.Prompt(),
		project:   prime.Project(),
		auth:      prime.Auth(),
		config:    prime.Config(),
		analytics: prime.Analytics(),
		svcModel:  prime.SvcModel(),
	}
}

// Run executes the add behavior.
func (a *Add) Run(ps AddRunParams) error {
	logging.Debug("Execute platforms add")

	params, err := prepareParams(ps.Params)
	if err != nil {
		return err
	}

	if a.project == nil {
		return locale.NewInputError("err_no_project")
	}

	if err := requirements.ExecuteRequirementOperation(&requirements.RequirementOperationParams{
		Output:              a.output,
		Prompt:              a.prompt,
		Project:             a.project,
		Auth:                a.auth,
		Config:              a.config,
		Analytics:           a.analytics,
		SvcModel:            a.svcModel,
		RequirementName:     params.name,
		RequirementVersion:  params.version,
		RequirementBitWidth: params.BitWidth,
		Operation:           model.OperationAdded,
		NsType:              model.NamespaceLanguage,
	}); err != nil {
		return locale.WrapError(err, "err_add_platform", "Could not add platform.")
	}

	a.output.Notice(locale.Tr("platform_added", params.name, params.version))

	return nil
}
