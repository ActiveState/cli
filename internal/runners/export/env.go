package export

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/target"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Env struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
	out       output.Outputer
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	auth      *authentication.Auth
	project   *project.Project
	cfg       *config.Instance
}

func NewEnv(prime primeable) *Env {
	return &Env{
		prime,
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

	rt, err := runtime_runbit.Update(e.prime, target.TriggerActivate)
	if err != nil {
		return locale.WrapError(err, "err_export_new_runtime", "Could not initialize runtime")
	}

	envVars := rt.Env().Variables

	e.out.Print(output.Prepare(envVars, envVars))

	return nil
}
