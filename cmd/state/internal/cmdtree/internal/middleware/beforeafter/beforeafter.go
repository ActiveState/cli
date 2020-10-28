package beforeafter

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/runners/run"
	"github.com/ActiveState/cli/pkg/project"
)

type BeforeAfter struct {
	Project *project.Project
}

func New(p *project.Project) *BeforeAfter {
	return &BeforeAfter{
		Project: p,
	}
}

func (ba *BeforeAfter) Wrap(next captain.ExecuteFunc) captain.ExecuteFunc {
	return func(cmd *captain.Command, args []string) error {
		runEvent := run.NewEvent(ba.Project.Events(), cmd.UseFull())

		if err := runEvent.Run(project.BeforeCmd); err != nil {
			return err // TODO: ctx
		}

		if err := next(cmd, args); err != nil {
			return err // TODO: ctx
		}

		if err := runEvent.Run(project.AfterCmd); err != nil {
			return err // TODO: ctx
		}

		return nil
	}
}
