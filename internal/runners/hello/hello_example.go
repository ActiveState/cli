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

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/example"
	"github.com/ActiveState/cli/pkg/platform/authentication"
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
	auth    *authentication.Auth
}

// New contains the scope in which an instance of Hello is constructed from an
// implementation of primeable.
func New(p primeable) *Hello {
	return &Hello{
		out:     p.Output(),
		auth:    p.Auth(),
	}
}

// rationalizeError is used to interpret the returned error and rationalize it for the end-user.
// This is so that end-users always get errors that clearly relate to what they were doing, with a good sense on what
// they can do to address it.
func rationalizeError(err *error) {
	var errNoNameProvided *example.NoNameProvidedError

	switch {
	case err == nil:
		return
	case errors.As(*err, &errNoNameProvided):
		// Errors that we are looking for should be wrapped in a user-facing error.
		// Ensure we wrap the top-level error returned from the runner and not
		// the unpacked error that we are inspecting.
		*err = errs.WrapUserFacing(*err, locale.Tl("hello_err_no_name", "Cannot say hello because no name was provided."))
	}
}

// Run contains the scope in which the hello runner logic is executed.
func (h *Hello) Run(params *Params) (rerr error) {
	defer rationalizeError(&rerr)

	h.out.Print(locale.Tl("hello_notice", "This command is for example use only"))

	// Reusable runner logic is contained within the runbits package.
	// You should only use this if you intend to share logic between
	// runners. Runners should NEVER invoke other runners.
	if err := example.SayHello(h.out, params.Name); err != nil {
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

	h.out.Print(locale.Tl(
		"hello_extra_info",
		"You are on commit {{.V0}}",
		constants.RevisionHashShort,
	))

	return nil
}