// +build !external

package main

import (
	"github.com/ActiveState/cli/internal/expander"
	"github.com/ActiveState/cli/internal/logging"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/state/activate"
	"github.com/ActiveState/cli/state/auth"
	"github.com/ActiveState/cli/state/events"
	"github.com/ActiveState/cli/state/keypair"
	"github.com/ActiveState/cli/state/new"
	"github.com/ActiveState/cli/state/organizations"
	"github.com/ActiveState/cli/state/projects"
	"github.com/ActiveState/cli/state/run"
	"github.com/ActiveState/cli/state/scripts"
	"github.com/ActiveState/cli/state/shim"
	"github.com/ActiveState/cli/state/show"
	"github.com/ActiveState/cli/state/update"
	"github.com/ActiveState/cli/state/variables"
)

// register will register any commands and expanders
func register() {
	logging.Debug("register")

	secretsapi.InitializeClient()

	Command.Append(activate.Command)
	Command.Append(events.Command)
	Command.Append(update.Command)
	Command.Append(auth.Command)
	Command.Append(organizations.Command)
	Command.Append(projects.Command)
	Command.Append(new.Command)
	Command.Append(show.Command)
	Command.Append(run.Command)
	Command.Append(scripts.Command)
	Command.Append(shim.Command)

	Command.Append(variables.NewCommand(secretsapi.Get()).Config())
	Command.Append(keypair.Command)

	expander.RegisterExpander("variables", expander.NewVarPromptingExpander(secretsapi.Get()))
}
