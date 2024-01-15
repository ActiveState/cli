package refresh

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/findproject"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
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
}

type Refresh struct {
	auth      *authentication.Auth
	prompt    prompt.Prompter
	out       output.Outputer
	svcModel  *model.SvcModel
	config    *config.Instance
	analytics analytics.Dispatcher
}

func New(prime primeable) *Refresh {
	return &Refresh{
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
		if errs.Matches(err, &projectfile.ErrorNoDefaultProject{}) {
			return locale.WrapError(err, "err_use_default_project_does_not_exist")
		}
		return locale.WrapError(err, "err_refresh_cannot_load_project", "Cannot load project to update runtime for")
	}

	rti, err := runtime.NewFromProject(proj, target.TriggerRefresh, r.analytics, r.svcModel, r.out, r.auth, r.config)
	if err != nil {
		return locale.WrapInputError(err, "err_refresh_runtime_new", "Could not update runtime for this project.")
	}

	execDir := setup.ExecDir(rti.Target().Dir())
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
