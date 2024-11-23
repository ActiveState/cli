package cmdcall

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events/cmdcall"
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

func (c *CmdCall) OnExecStart(cmd *captain.Command, _ []string) error {
	cc := cmdcall.New(c.primer, cmd.JoinedSubCommandNames())
	if err := cc.Run(project.BeforeCmd); err != nil {
		return errs.Wrap(err, "before-command event run failure")
	}
	return nil
}

func (c *CmdCall) OnExecStop(cmd *captain.Command, _ []string) error {
	cc := cmdcall.New(c.primer, cmd.JoinedSubCommandNames())
	if err := cc.Run(project.AfterCmd); err != nil {
		return errs.Wrap(err, "after-command event run failure")
	}
	return nil
}
