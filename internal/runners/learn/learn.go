package learn

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/skratchdot/open-golang/open"
)

const cheetSheetURL = "https://platform.activestate.com/state-tool-cheat-sheet"

type Learn struct {
	out output.Outputer
}

type primeable interface {
	primer.Outputer
}

func New(prime primeable) *Learn {
	return &Learn{prime.Output()}
}

func (l *Learn) Run() error {
	err := open.Run(cheetSheetURL)
	if err != nil {
		return locale.WrapError(err, "err_learn_open", "Could not open State Tool cheat sheet in the browser")
	}

	l.out.Print(locale.Tl("learn_info", "If a browser window doesn't automatically open, visit: [ACTIONABLE]{{.V0}}[/RESET] to learn about common commands", cheetSheetURL))
	return nil
}
