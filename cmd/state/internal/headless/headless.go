package headless

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
)

// Notify will output a message to users when the project is in a headless
// state and no error is encountered. If a cmd name is provided, it's
// particular headless message is outputted immediately before the general
// message.
func Notify(prime *primer.Values, err error, cmdNames ...string) {
	if err != nil || !prime.Project().IsHeadless() {
		return
	}

	for _, cmd := range cmdNames {
		prime.Output().Error(locale.T("message_headless_" + cmd))
	}

	prime.Output().Error(locale.T("message_headless"))
}
