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

func (i *Info) Run(params *InfoRunParams) error {
	if err := runbits.SayHello(i.out, i.auth.WhoAmI()); err != nil {
		err = locale.WrapError(
			err, "hello_info_err_say", "Cannot say hello without a name",
		)
		if !i.auth.Authenticated() {
			err = errs.AddTips(err, locale.Tl("hello_info_suggest_login", "Try logging in"))
		}
		return err
	}

	projMsg := locale.Tl(
		"hello_info_warn_no_project", "Not in a project directory.",
	)
	if i.pj != nil {
		projMsg = locale.Tl(
			"hello_info_project",
			"Project: {{.V0}}",
			i.pj.Namespace().String(),
		)
	}
	i.out.Print(projMsg)

	if params.Extra {
		commitMsg, err := currentCommitMessage(i.pj)
		if err != nil {
			err = locale.WrapError(
				err, "hello_info_err_get_commit_msg", " Cannot get commit message",
			)
			return errs.AddTips(err, locale.Tl(
				"hello_info_suggest_ensure_commit", "Ensure project has commits",
			))
		}

		i.out.Print(locale.Tl(
			"hello_info_extra",
			"Current commit message: {{.V0}}",
			commitMsg,
		))
	}

	return nil
}

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
