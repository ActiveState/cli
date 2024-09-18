package refresh

import (
	"errors"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/findproject"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/cli/pkg/runtime_helpers"
)

type Params struct {
	Namespace *project.Namespaced
}

type primeable interface {
	primer.Auther
	primer.Prompter
	primer.Outputer
	primer.Configurer
	primer.SvcModeler
	primer.Analyticer
	primer.Projecter
}

type Refresh struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
	auth      *authentication.Auth
	prompt    prompt.Prompter
	out       output.Outputer
	svcModel  *model.SvcModel
	config    *config.Instance
	analytics analytics.Dispatcher
}

func New(prime primeable) *Refresh {
	return &Refresh{
		prime,
		prime.Auth(),
		prime.Prompt(),
		prime.Output(),
		prime.SvcModel(),
		prime.Config(),
		prime.Analytics(),
	}
}

func (r *Refresh) Run(params *Params) error {
	logging.Debug("Refresh %v", params.Namespace)

	proj, err := findproject.FromInputByPriority("", params.Namespace, r.config, r.prompt)
	if err != nil {
		var errNoDefaultProject *projectfile.ErrorNoDefaultProject
		if errors.As(err, &errNoDefaultProject) {
			return locale.WrapError(err, "err_use_default_project_does_not_exist")
		}
		return rationalize.ErrNoProject
	}

	r.prime.SetProject(proj)

	r.out.Notice(locale.Tr("operating_message", proj.NamespaceString(), proj.Dir()))

	needsUpdate, err := runtime_helpers.NeedsUpdate(proj, nil)
	if err != nil {
		return errs.Wrap(err, "could not determine if runtime needs update")
	}

	if !needsUpdate {
		return locale.NewInputError("refresh_runtime_uptodate")
	}

	rti, err := runtime_runbit.Update(r.prime, trigger.TriggerRefresh, runtime_runbit.WithoutHeaders(), runtime_runbit.WithIgnoreAsync())
	if err != nil {
		return locale.WrapError(err, "err_refresh_runtime_new", "Could not update runtime for this project.")
	}

	execDir := rti.Env(false).ExecutorsPath
	r.out.Print(output.Prepare(
		locale.Tr("refresh_project_statement", proj.NamespaceString(), proj.Dir(), execDir),
		&struct {
			Namespace   string `json:"namespace"`
			Path        string `json:"path"`
			Executables string `json:"executables"`
		}{
			proj.NamespaceString(),
			proj.Dir(),
			execDir,
		}))

	return nil
}
