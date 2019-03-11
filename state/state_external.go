// +build external

package main

import (
	"github.com/ActiveState/cli/internal/logging"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/state/activate"
	"github.com/ActiveState/cli/state/auth"
	"github.com/ActiveState/cli/state/update"
)

// register will register any commands and expanders
func register() {
	logging.Debug("register external")

	secretsapi.InitializeClient()

	Command.Append(activate.Command)
	Command.Append(update.Command)
	Command.Append(auth.Command)
}
