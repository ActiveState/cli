package eval

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
)

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Projecter
	primer.SvcModeler
}

type Params struct {
	Target string
}

type Eval struct {
	prime primeable
}

func New(p primeable) *Eval {
	return &Eval{
		prime: p,
	}
}

func (e *Eval) Run(params *Params) (rerr error) {
	defer rationalizeError(&rerr)

	out := e.prime.Output()
	auth := e.prime.Auth()
	proj := e.prime.Project()

	out.Notice(output.Title(locale.Tl("title_eval", "Evaluating target: {{.V0}}", params.Target)))

	if !auth.Authenticated() {
		return rationalize.ErrNotAuthenticated
	}

	if proj == nil {
		return rationalize.ErrNoProject
	}

	commitID, err := localcommit.Get(proj.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get commit ID")
	}

	pg := output.StartSpinner(out, locale.Tl("progress_eval", "Evaluating ... "), constants.TerminalAnimationInterval)
	defer func() {
		if pg != nil {
			pg.Stop(locale.T("progress_fail") + "\n")
		}
	}()

	bp := buildplanner.NewBuildPlannerModel(auth, e.prime.SvcModel())
	if err := bp.BuildTarget(proj.Owner(), proj.Name(), commitID.String(), params.Target); err != nil {
		return locale.WrapError(err, "err_eval", "Failed to evaluate target '{{.V0}}'", params.Target)
	}

	if err := bp.WaitForBuild(commitID, proj.Owner(), proj.Name(), &params.Target); err != nil {
		return locale.WrapError(err, "err_eval_wait_for_build", "Failed to build target: '{{.V)}}'", params.Target)
	}

	pg.Stop("OK")
	pg = nil

	out.Notice(locale.Tl("notice_eval_success", "Target successfully evaluated"))

	return nil
}
