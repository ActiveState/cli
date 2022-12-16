package learn

import (
	"github.com/ActiveState/cli/internal-as/constants"
	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/internal-as/logging"
	"github.com/ActiveState/cli/internal-as/output"
	"github.com/ActiveState/cli/internal-as/primer"
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
	l.out.Print(locale.Tl("learn_info", "Opening [ACTIONABLE]{{.V0}}[/RESET] in browser", constants.CheatSheetURL))

	err := open.Run(constants.CheatSheetURL)
	if err != nil {
		logging.Warning("Could not open browser: %v", err)
		l.out.Notice(locale.Tr("err_browser_open", constants.CheatSheetURL))
	}

	return nil
}
