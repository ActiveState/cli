package headless

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/project"
)

// Notify will output a message to users when the project is in a headless
// state and no error is encountered. If a cmd name is provided, it's
// particular headless message is outputted immediately before the general
// message.
func Notify(out output.Outputer, proj *project.Project, err error, cmdNames ...string) {
	if err != nil || true /*!proj.IsHeadless()*/ {
		return
	}

	for _, cmd := range cmdNames {
		out.Notice(locale.T("message_headless_" + cmd))
	}

	out.Notice(locale.T("message_headless"))
}
