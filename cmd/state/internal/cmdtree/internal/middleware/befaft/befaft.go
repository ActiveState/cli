package befaft

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/runners/run"
	"github.com/ActiveState/cli/pkg/project"
)

type BefAft struct {
	Project *project.Project
}

func New(p *project.Project) *BefAft {
	return &BefAft{
		Project: p,
	}
}

func (ba *BefAft) Wrap(next captain.ExecuteFunc) captain.ExecuteFunc {
	return func(cmd *captain.Command, args []string) error {
		befEvent, err := run.NewEvent(ba.Project.Events(), project.BeforeCmd)
		if err != nil {
			return err // TODO: ctx
		}

		if err := befEvent.Run(); err != nil {
			return err // TODO: ctx
		}

		if err := next(cmd, args); err != nil {
			return err // TODO: ctx
		}

		aftEvent, err := run.NewEvent(ba.Project.Events(), project.AfterCmd)
		if err != nil {
			return err // TODO: ctx
		}

		if err := aftEvent.Run(); err != nil {
			return err // TODO: ctx
		}

		return nil
	}
}
