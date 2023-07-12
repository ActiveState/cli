package hello

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type infoPrimeable interface {
	primer.Outputer
	primer.Auther
	primer.Projecter
}

type InfoRunParams struct {
	Extra bool
}

type Info struct {
	out  output.Outputer
	auth *authentication.Auth
	pj   *project.Project
}

func NewInfo(p infoPrimeable) *Info {
	return &Info{
		out:  p.Output(),
		auth: p.Auth(),
		pj:   p.Project(),
	}
}

// Run contains the scope in which the info runner logic is executed. It can be
// helpful to group stages of the runner logic into subhandlers/seclusions so
// that the behavior of the runner is immediately apparent.
func (i *Info) Run(params *InfoRunParams) error {
	if err := runOutputHello(i.out, i.auth); err != nil {
		// Because the "run" subhandlers are purely for seculding
		// logic, it is not necessarily helpful to add localized error
		// context.
		return err
	}

	runOutputProjectMessage(i.out, i.pj)

	return runOutputExtra(i.out, i.pj, params)
}

func runOutputHello(out output.Outputer, auth *authentication.Auth) error {
	if err := runbits.SayHello(out, auth.WhoAmI()); err != nil {
		// As in the hello runner, errors should nearly always be
		// localized at this level.
		err = locale.WrapError(
			err, "hello_info_err_say", "Cannot say hello without a name",
		)
		// It's also useful to offer users reasonable tips on recourses.
		if !auth.Authenticated() {
			err = errs.AddTips(
				err, locale.Tl("hello_info_suggest_login", "Try logging in"),
			)
		}
		return err
	}

	return nil
}

func runOutputProjectMessage(out output.Outputer, pj *project.Project) {
	projMsg := locale.Tl(
		"hello_info_warn_no_project", "Not in a project directory.",
	)
	if pj != nil {
		projMsg = locale.Tl(
			"hello_info_project",
			"Project: {{.V0}}",
			pj.Namespace().String(),
		)
	}

	out.Print(projMsg)
}

func runOutputExtra(out output.Outputer, pj *project.Project, params *InfoRunParams) error {
	if !params.Extra {
		return nil
	}

	commitMsg, err := currentCommitMessage(pj)
	if err != nil {
		err = locale.WrapError(
			err, "hello_info_err_get_commit_msg", " Cannot get commit message",
		)
		return errs.AddTips(
			err,
			locale.Tl("hello_info_suggest_ensure_commit", "Ensure project has commits"),
		)
	}

	out.Print(locale.Tl(
		"hello_info_extra",
		"Current commit message: {{.V0}}",
		commitMsg,
	))

	return nil
}

// currentCommitMessage contains the scope in which the current commit message
// is obtained. Since it is a sort of construction function that has some
// complexity, it is helpful to provide localized error context. Secluding this
// sort of logic is helpful to keep the subhandlers clean.
func currentCommitMessage(pj *project.Project) (string, error) {
	if pj == nil || pj.CommitUUID() == "" {
		return "", errs.New("Cannot determine which commit UUID to use")
	}

	commit, err := model.GetCommit(pj.CommitUUID())
	if err != nil {
		return "", locale.NewError(
			"hello_info_err_get_commitr", "Cannot get commit from server",
		)
	}

	commitMsg := locale.Tl("hello_info_warn_no_commit", "Commit description not provided.")
	if commit.Message != "" {
		commitMsg = commit.Message
	}

	return commitMsg, nil
}
