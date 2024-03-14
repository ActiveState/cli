// Package hello provides "hello command" logic and associated types. By
// convention, the construction function for the primary type (also named
// "hello") is simply named "New". For other construction functions, the name
// of the type is used as a suffix. For instance, "hello.NewInfo()" for the
// type "hello.Info". Similarly, "Info" would be used as a prefix or suffix for
// the info-related types like "InfoRunParams". Each "group of types" is usually
// expressed in its own file.
package hello

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// primeable describes the app-level dependencies that a runner will need.
type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Projecter
}

// Params defines the parameters needed to execute a given runner. These
// values are typically collected from flags and arguments entered into the
// cli, but there is no reason that they couldn't be set in another manner.
type Params struct {
	Name  string
	Echo  Text
	Extra bool
}

// NewParams contains a scope in which default or construction-time values
// can be set. If no default or construction-time values are necessary, direct
// construction of Params is fine, and this construction func may be dropped.
func NewParams() *Params {
	return &Params{}
}

// Hello defines the app-level dependencies that are accessible within the Run
// function.
type Hello struct {
	out     output.Outputer
	project *project.Project
	auth    *authentication.Auth
}

// New contains the scope in which an instance of Hello is constructed from an
// implementation of primeable.
func New(p primeable) *Hello {
	return &Hello{
		out:     p.Output(),
		project: p.Project(),
		auth:    p.Auth(),
	}
}

// rationalizeError is used to interpret the returned error and rationalize it for the end-user.
// This is so that end-users always get errors that clearly relate to what they were doing, with a good sense on what
// they can do to address it.
func rationalizeError(err *error) {
	switch {
	case err == nil:
		return
	case errs.Matches(*err, &runbits.NoNameProvidedError{}):
		// Errors that we are looking for should be wrapped in a user-facing error.
		// Ensure we wrap the top-level error returned from the runner and not
		// the unpacked error that we are inspecting.
		*err = errs.WrapUserFacing(*err, locale.Tl("hello_err_no_name", "Cannot say hello because no name was provided."))
	case errors.Is(*err, rationalize.ErrNoProject):
		// It's useful to offer users reasonable tips on recourses.
		*err = errs.WrapUserFacing(
			*err,
			locale.Tl("hello_err_no_project", "Cannot say hello because you are not in a project directory."),
			errs.SetTips(
				locale.Tl("hello_suggest_checkout", "Try using '[ACTIONABLE]state checkout[/RESET]' first."),
			),
		)
	}
}

// Run contains the scope in which the hello runner logic is executed.
func (h *Hello) Run(params *Params) (rerr error) {
	defer rationalizeError(&rerr)

	h.out.Print(locale.Tl("hello_notice", "This command is for example use only"))

	if h.project == nil {
		return rationalize.ErrNoProject
	}

	// Reusable runner logic is contained within the runbits package.
	// You should only use this if you intend to share logic between
	// runners. Runners should NEVER invoke other runners.
	if err := runbits.SayHello(h.out, params.Name); err != nil {
		// Errors should nearly always be localized.
		return errs.Wrap(
			err, "Cannot say hello.",
		)
	}

	if params.Echo.IsSet() {
		h.out.Print(locale.Tl(
			"hello_echo_msg", "Echoing: {{.V0}}",
			params.Echo.String(),
		))
	}

	if !params.Extra {
		return nil
	}

	// Grab data from the platform.
	commitMsg, err := currentCommitMessage(h.project, h.auth)
	if err != nil {
		err = errs.Wrap(
			err, "Cannot get commit message",
		)
		return errs.AddTips(
			err,
			locale.Tl("hello_info_suggest_ensure_commit", "Ensure project has commits"),
		)
	}

	h.out.Print(locale.Tl(
		"hello_extra_info",
		"Project: {{.V0}}\nCurrent commit message: {{.V1}}",
		h.project.Namespace().String(), commitMsg,
	))

	return nil
}

// currentCommitMessage contains the scope in which the current commit message
// is obtained. Since it is a sort of construction function that has some
// complexity, it is helpful to provide localized error context. Secluding this
// sort of logic is helpful to keep the subhandlers clean.
func currentCommitMessage(proj *project.Project, auth *authentication.Auth) (string, error) {
	if proj == nil {
		return "", errs.New("Cannot determine which project to use")
	}

	commitId, err := localcommit.Get(proj.Dir())
	if err != nil {
		return "", errs.Wrap(err, "Cannot determine which commit to use")
	}

	commit, err := model.GetCommit(commitId, auth)
	if err != nil {
		return "", errs.Wrap(err, "Cannot get commit from server")
	}

	commitMsg := locale.Tl("hello_info_warn_no_commit", "Commit description not provided.")
	if commit.Message != "" {
		commitMsg = commit.Message
	}

	return commitMsg, nil
}
