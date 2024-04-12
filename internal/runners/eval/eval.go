package eval

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
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
	defer rationalizeError(&rerr)

	e.out.Notice(output.Title(locale.Tl("title_eval", "Evaluating target: {{.V0}}", params.Target)))

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

	pg := output.StartSpinner(e.out, locale.Tl("progress_eval", "Evaluating ... "), constants.TerminalAnimationInterval)
	defer func() {
		if pg != nil {
			pg.Stop(locale.T("progress_fail") + "\n")
		}
	}()

	bp := model.NewBuildPlannerModel(e.auth)
	if _, _, err := bp.FetchBuildResult(commitID, e.project.Owner(), e.project.Name(), &params.Target); err != nil {
		return locale.WrapError(err, "err_eval", "Failed to evaluate target '{{.V0}}'", params.Target)
	}

	pg.Stop("OK")
	pg = nil

	e.out.Notice(locale.Tl("notice_eval_success", "Target successfully evaluated"))

	return nil
}
