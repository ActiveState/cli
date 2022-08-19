package export

import (
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/errs"
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
}

func NewEnv(prime primeable) *Env {
	return &Env{
		prime.Output(),
		prime.Analytics(),
		prime.SvcModel(),
		prime.Auth(),
		prime.Project(),
	}
}

func (e *Env) Run() error {
	rt, _, err := runtime.NewFromProject(e.project, target.TriggerActivate, e.analytics, e.svcModel, e.out, e.auth)
	if err != nil {
		if errs.Matches(err, &model.ErrNoMatchingPlatform{}) {
			branches, err := model.BranchNamesForProjectFiltered(e.project.Owner(), e.project.Name(), e.project.BranchName())
			if err == nil && len(branches) > 1 {
				return locale.NewInputError("err_alternate_branches", "", e.project.BranchName(), strings.Join(branches, "\n - "))
			}
		}
		if !authentication.LegacyGet().Authenticated() {
			return locale.WrapError(err, "err_export_env_auth", "Could not update runtime files. If this is a private project ensure that you are authenticated.")
		}
		return locale.WrapError(err, "err_export_new_runtime", "Could not get new runtime")
	}

	env, err := rt.Env(false, true)
	if err != nil {
		return locale.WrapError(err, "err_env_get_env", "Could not get runtime environment")
	}

	for k, v := range env {
		e.out.Print((locale.Tl("env_output_env", "{{.V0}}: {{.V2}}", k, v)))
	}

	return nil
}
