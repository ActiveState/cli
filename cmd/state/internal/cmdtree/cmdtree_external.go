// +build external

package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/logging"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/state/auth"
	"github.com/ActiveState/cli/state/events"
	"github.com/ActiveState/cli/state/export"
	"github.com/ActiveState/cli/state/fork"
	"github.com/ActiveState/cli/state/organizations"
	"github.com/ActiveState/cli/state/projects"
	"github.com/ActiveState/cli/state/pull"
	"github.com/ActiveState/cli/state/run"
	"github.com/ActiveState/cli/state/scripts"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/ActiveState/cli/state/show"
	"github.com/ActiveState/cli/state/update"
)

// applyLegacyChildren will register any commands and expanders
func applyLegacyChildren(cmd *captain.Command, globals *globalOptions) {
	logging.Debug("register external")

	secretsapi.InitializeClient()

	setLegacyOutput(globals)

	cmd.AddLegacyChildren(
		events.Command,
		update.Command,
		auth.Command,
		organizations.Command,
		projects.Command,
		show.Command,
		run.Command,
		scripts.Command,
		pull.Command,
		export.Command,
		secrets.NewCommand(secretsapi.Get(), &globals.Output).Config(),
		fork.Command,
	)
}

func setLegacyOutput(globals *globalOptions) {
	auth.Flags.Output = &globals.Output
	organizations.Flags.Output = &globals.Output
	scripts.Flags.Output = &globals.Output
	secrets.Flags.Output = &globals.Output
}
