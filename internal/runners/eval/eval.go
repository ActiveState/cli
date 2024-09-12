package eval

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	buildscript_runbit "github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Projecter
	primer.Configurer
}

type Params struct {
	Target string
}

type Eval struct {
	out     output.Outputer
	project *project.Project
	auth    *authentication.Auth
	cfg     *config.Instance
}

func New(p primeable) *Eval {
	return &Eval{
		out:     p.Output(),
		project: p.Project(),
		auth:    p.Auth(),
		cfg:     p.Config(),
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

	commitID, err := buildscript_runbit.CommitID(e.project.Dir(), e.cfg)
	if err != nil {
		return errs.Wrap(err, "Unable to get commit ID")
	}

	pg := output.StartSpinner(e.out, locale.Tl("progress_eval", "Evaluating ... "), constants.TerminalAnimationInterval)
	defer func() {
		if pg != nil {
			pg.Stop(locale.T("progress_fail") + "\n")
		}
	}()

	bp := buildplanner.NewBuildPlannerModel(e.auth)
	if err := bp.BuildTarget(e.project.Owner(), e.project.Name(), commitID.String(), params.Target); err != nil {
		return locale.WrapError(err, "err_eval", "Failed to evaluate target '{{.V0}}'", params.Target)
	}

	if err := bp.WaitForBuild(commitID, e.project.Owner(), e.project.Name(), &params.Target); err != nil {
		return locale.WrapError(err, "err_eval_wait_for_build", "Failed to build target: '{{.V)}}'", params.Target)
	}

	pg.Stop("OK")
	pg = nil

	e.out.Notice(locale.Tl("notice_eval_success", "Target successfully evaluated"))

	return nil
}
