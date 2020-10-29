package beforeafter

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/run"
	"github.com/ActiveState/cli/pkg/project"
)

// BeforeAfter manages the before/after command event intercept scope.
type BeforeAfter struct {
	primer *primer.Values
}

// New returns a pointer to a prepared BeforeAfter instance.
func New(p *primer.Values) *BeforeAfter {
	return &BeforeAfter{
		primer: p,
	}
}

// InterceptExec handles the before command logic, calls the next ExecuteFunc,
// and then handles the after command logic.
func (ba *BeforeAfter) InterceptExec(next captain.ExecuteFunc) captain.ExecuteFunc {
	return func(cmd *captain.Command, args []string) error {
		runEvent := run.NewEvent(ba.primer, cmd.UseFull())

		if err := runEvent.Run(project.BeforeCmd); err != nil {
			return errs.Wrap(err, "before-command event run failure")
		}

		if err := next(cmd, args); err != nil {
			return errs.Wrap(err, "before/after-command next func failure")
		}

		if err := runEvent.Run(project.AfterCmd); err != nil {
			return errs.Wrap(err, "after-command event run failure")
		}

		return nil
	}
}
