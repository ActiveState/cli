// Package hello provides "hello command" logic and associated types. By
// convention, the construction function for the primary type (also named
// "hello") is simply named "New". For other construction functions, the name
// of the type is used as a suffix. For instance, "hello.NewInfo()" for the
// type "hello.Info". Similarly, "Info" would be used as a prefix or suffix for
// the info-related types like "InfoRunParams". Each "group of types" is usually
// expressed in its own file.
package hello

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// primeable describes the app-level dependencies that a runner will need.
type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Projecter
}

// RunParams defines the parameters needed to execute a given runner. These
// values are typically collected from flags and arguments entered into the
// cli, but there is no reason that they couldn't be set in another manner.
type RunParams struct {
	Name  string
	Extra bool
}

// NewRunParams contains a scope in which default or construction-time values
// can be set. If no default or construction-time values are necessary, direct
// construction of RunParams is fine, and this construction func may be dropped.
func NewRunParams() *RunParams {
	return &RunParams{
		Name: "Friend",
	}
}

// Hello defines the app-level dependencies that are accessible within the Run
// function.
type Hello struct {
	out output.Outputer
	pj  *project.Project
}

// New contains the scope in which an instance of Hello is constructed from an
// implementation of primeable.
func New(p primeable) *Hello {
	return &Hello{
		out: p.Output(),
		pj:  p.Project(),
	}
}

// Run contains the scope in which the hello runner logic is executed.
func (h *Hello) Run(params *RunParams) error {
	// Reusable runner logic is contained within the runbits package.
	// You should only use this if you intend to share logic between
	// runners. Runners should NEVER invoke other runners.
	if err := runbits.SayHello(h.out, params.Name); err != nil {
		// Errors should nearly always be localized.
		return locale.WrapError(
			err, "hello_cannot_say", "Cannot say hello.",
		)
	}

	if !params.Extra {
		return nil
	}

	if h.pj == nil {
		err := locale.NewInputError(
			"hello_info_err_no_project", "Not in a project directory.",
		)

		// It's useful to offer users reasonable tips on recourses.
		return errs.AddTips(err, locale.Tl(
			"hello_suggest_checkout",
			"Try using [ACTIONABLE]`state checkout`[/RESET] first.",
		))
	}

	// Grab data from the platform.
	commitMsg, err := currentCommitMessage(h.pj)
	if err != nil {
		err = locale.WrapError(
			err, "hello_info_err_get_commit_msg", " Cannot get commit message",
		)
		return errs.AddTips(
			err,
			locale.Tl("hello_info_suggest_ensure_commit", "Ensure project has commits"),
		)
	}

	h.out.Print(locale.Tl(
		"hello_extra_info",
		"Project: {{.V0}}\nCurrent commit message: {{.V1}}",
		h.pj.Namespace().String(), commitMsg,
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
