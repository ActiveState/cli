// +build !external

package main

import (
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/state/activate"
	"github.com/ActiveState/cli/state/auth"
	"github.com/ActiveState/cli/state/events"
	"github.com/ActiveState/cli/state/export"
	"github.com/ActiveState/cli/state/keypair"
	"github.com/ActiveState/cli/state/new"
	"github.com/ActiveState/cli/state/organizations"
	pkg "github.com/ActiveState/cli/state/package"
	"github.com/ActiveState/cli/state/projects"
	"github.com/ActiveState/cli/state/pull"
	"github.com/ActiveState/cli/state/run"
	"github.com/ActiveState/cli/state/scripts"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/ActiveState/cli/state/show"
	"github.com/ActiveState/cli/state/update"
)

var registerMessage = "register"

var commandList = []*commands.Command{
	activate.Command,
	events.Command,
	update.Command,
	auth.Command,
	organizations.Command,
	projects.Command,
	new.Command,
	show.Command,
	run.Command,
	scripts.Command,
	pull.Command,
	export.Command,
	pkg.Command,
	secrets.NewCommand(secretsapi.Get()).Config(),
	keypair.Command,
}
