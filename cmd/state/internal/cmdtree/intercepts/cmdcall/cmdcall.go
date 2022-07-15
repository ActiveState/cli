package cmdcall

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events/cmdcall"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/svcctl"
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
		sockPath := svcctl.NewIPCSockPathFromGlobals().String()
		cc := cmdcall.New(c.primer, sockPath, cmd.UseFull())

		if err := cc.Run(project.BeforeCmd); err != nil {
			return errs.Wrap(err, "before-command event run failure")
		}

		if err := next(cmd, args); err != nil {
			return err
		}

		if err := cc.Run(project.AfterCmd); err != nil {
			return errs.Wrap(err, "after-command event run failure")
		}

		return nil
	}
}
