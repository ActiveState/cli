package cmdcall

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events/cmdcall"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/project"
)

// CmdCall manages the event handling flow triggered by command calls.
type CmdCall struct {
	primer *primer.Values
}

// New returns a pointer to a prepared CmdCall instance.
func New(p *primer.Values) *CmdCall {
	return &CmdCall{
		primer: p,
	}
}

// InterceptExec handles the before command logic, calls the next ExecuteFunc,
// and then handles the after command logic.
func (c *CmdCall) InterceptExec(next captain.ExecuteFunc) captain.ExecuteFunc {
	return func(cmd *captain.Command, args []string) error {
		cc := cmdcall.New(c.primer, cmd.UseFull())

		if err := cc.Run(project.BeforeCmd); err != nil {
			return errs.Wrap(err, "before-command event run failure")
		}

		if err := next(cmd, args); err != nil {
			// check can be removed when no runners return failures and,
			// possibly, when main pkg error handling digs through the error
			// chain to get to nested failures
			if fail, ok := err.(*failures.Failure); ok {
				if fail == nil {
					return nil
				}
				return fail // wrapping would break error handling higher in stack
			}
			return errs.Wrap(err, "before/after-command next func failure")
		}

		if err := cc.Run(project.AfterCmd); err != nil {
			return errs.Wrap(err, "after-command event run failure")
		}

		return nil
	}
}
