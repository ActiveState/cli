package export

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
)

type Env struct {
	out       output.Outputer
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	auth      *authentication.Auth
	project   *project.Project
	cfg       *config.Instance
}

func NewEnv(prime primeable) *Env {
	return &Env{
		prime.Output(),
		prime.Analytics(),
		prime.SvcModel(),
		prime.Auth(),
		prime.Project(),
		prime.Config(),
	}
}

func (e *Env) Run() error {
	if e.project == nil {
		return locale.NewInputError("err_env_no_project", "No project found.")
	}

	e.out.Notice(locale.Tr("export_project_statement",
		e.project.NamespaceString(),
		e.project.Dir()),
	)

	rt, err := runtime.SolveAndUpdate(e.auth, e.out, e.analytics, e.project, nil, target.TriggerActivate, e.svcModel, e.cfg, runtime.OptMinimalUI, false)
	if err != nil {
		return locale.WrapError(err, "err_export_new_runtime", "Could not initialize runtime")
	}

	env, err := rt.Env(false, true)
	if err != nil {
		return locale.WrapError(err, "err_env_get_env", "Could not get runtime environment")
	}

	e.out.Print(output.Prepare(env, env))

	return nil
}
