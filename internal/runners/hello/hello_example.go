// Package hello provides "hello command" logic and associated types. By
// convention, the construction function for the primary type (also named
// "hello") is simply named "New". For other construction functions, the name
// of the type is used as a suffix. For instance, "hello.NewInfo()" for the
// type "hello.Info". Similarly, "Info" would be used as a prefix or suffix for
// the info-related types like "InfoRunParams". Each "group of types" is usually
// expressed in its own file.
package hello

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
)

// primeable describes the app-level dependencies that a runner will need.
type primeable interface {
	primer.Outputer
}

// RunParams defines the parameters needed to execute a given runner. These
// values are typically collected from flags and arguments entered into the
// cli, but there is no reason that they couldn't be set in another manner.
type RunParams struct {
	Named string
}

// NewRunParams contains a scope in which default or construction-time values
// can be set. If no default or construction-time values are necessary, direct
// construction of RunParams is fine, and this construction func may be dropped.
func NewRunParams() *RunParams {
	return &RunParams{
		Named: "Friend",
	}
}

// Hello defines the app-level dependencies that are accessible within the Run
// function.
type Hello struct {
	out output.Outputer
}

// New contains the scope in which an instance of Hello is constructed from an
// implementation of primeable.
func New(p primeable) *Hello {
	return &Hello{
		out: p.Output(),
	}
}

// Run contains the scope in which the hello runner logic is executed.
func (h *Hello) Run(params *RunParams) error {
	// Reusable runner logic is contained within the runbits package.
	if err := runbits.SayHello(h.out, params.Named); err != nil {
		// Errors should nearly always be localized.
		return locale.WrapError(
			err, "hello_cannot_say", "Cannot say hello.",
		)
	}

	return nil
}
