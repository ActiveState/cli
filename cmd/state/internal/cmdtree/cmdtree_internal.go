// +build !external

package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/logging"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/state/events"
	"github.com/ActiveState/cli/state/fork"
	"github.com/ActiveState/cli/state/invite"
	"github.com/ActiveState/cli/state/pull"
	"github.com/ActiveState/cli/state/scripts"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/ActiveState/cli/state/show"
	"github.com/ActiveState/cli/state/update"
)

// applyLegacyChildren will register any commands and expanders
func applyLegacyChildren(cmd *captain.Command, globals *globalOptions) {
	logging.Debug("register")

	secretsapi.InitializeClient()

	setLegacyOutput(globals)

	cmd.AddLegacyChildren(
		events.Command,
		update.Command,
		show.Command,
		scripts.Command,
		pull.Command,
		invite.Command,
		secrets.NewCommand(secretsapi.Get(), &globals.Output).Config(),
		fork.Command,
	)
}
