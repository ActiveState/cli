package eval

import (
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Projecter
}

type Params struct {
	Target string
}

type Eval struct {
	out     output.Outputer
	project *project.Project
	auth    *authentication.Auth
}

func New(p primeable) *Eval {
	return &Eval{
		out:     p.Output(),
		project: p.Project(),
		auth:    p.Auth(),
	}
}

func (e *Eval) Run(params *Params) (rerr error) {
	defer rationalizeError(e.auth, e.project, &rerr)

	if !e.auth.Authenticated() {
		return rationalize.ErrNotAuthenticated
	}

	if e.project == nil {
		return rationalize.ErrNoProject
	}

	script, err := buildscript.NewScriptFromProject(e.project, e.auth)
	if err != nil {
		return errs.Wrap(err, "Could not get local build script")
	}

	var target string
	for _, assignment := range script.Assignments {
		if strings.EqualFold(assignment.Key, params.Target) {
			target = assignment.Key
		}
	}

	if target == "" {
		return locale.NewInputError("err_eval_target_not_found", "Target '{{.V0}}' not found", params.Target)
	}

	pg := output.StartSpinner(e.out, locale.Tl("progress_eval", "Evaluating ... "), constants.TerminalAnimationInterval)

	bp := model.NewBuildPlannerModel(e.auth)
	if err := bp.Evaluate(e.project.Owner(), e.project.Name(), target, script.Expr); err != nil {
		return locale.WrapError(err, "err_eval", "Failed to evaluate target '{{.V0}}'", target)
	}

	pg.Stop("OK")

	e.out.Notice(locale.Tl("notice_eval_success", "Target successfully evaluated"))

	return nil
}
