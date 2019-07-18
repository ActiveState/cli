// +build !external

package main

import (
	"github.com/ActiveState/cli/internal/logging"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/state/activate"
	"github.com/ActiveState/cli/state/auth"
	"github.com/ActiveState/cli/state/events"
	"github.com/ActiveState/cli/state/export"
	"github.com/ActiveState/cli/state/keypair"
	"github.com/ActiveState/cli/state/new"
	"github.com/ActiveState/cli/state/organizations"
	"github.com/ActiveState/cli/state/projects"
	"github.com/ActiveState/cli/state/pull"
	"github.com/ActiveState/cli/state/run"
	"github.com/ActiveState/cli/state/scripts"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/ActiveState/cli/state/show"
	"github.com/ActiveState/cli/state/update"
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
	Command.Append(pull.Command)
	Command.Append(export.Command)

	Command.Append(secrets.NewCommand(secretsapi.Get()).Config())
	Command.Append(keypair.Command)
}
