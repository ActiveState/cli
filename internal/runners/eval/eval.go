package eval

import (
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/localcommit"
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

type errTargetNotFound struct {
	error
	target string
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

	commitID, err := localcommit.Get(e.project.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get commit ID")
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
		return errTargetNotFound{target: params.Target}
	}

	pg := output.StartSpinner(e.out, locale.Tl("progress_eval", "Evaluating ... "), constants.TerminalAnimationInterval)

	bp := model.NewBuildPlannerModel(e.auth)
	if err := bp.BuildTarget(commitID.String(), target); err != nil {
		return locale.WrapError(err, "err_eval", "Failed to evaluate target '{{.V0}}'", target)
	}

	pg.Stop("OK")

	e.out.Notice(locale.Tl("notice_eval_success", "Target successfully evaluated"))

	return nil
}
