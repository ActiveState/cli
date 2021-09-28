package learn

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/skratchdot/open-golang/open"
)

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
	l.out.Print(locale.Tr("learn_info", "Opening [ACTIONABLE]{{.V0}}[/RESET] in browser", constants.CheatSheetURL))

	err := open.Run(constants.CheatSheetURL)
	if err != nil {
		return locale.WrapError(err, "err_learn_open", constants.CheatSheetURL)
	}

	return nil
}
