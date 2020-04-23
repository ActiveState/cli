package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/organizations"
)

func newOrganizationsCommand(globals *globalOptions) *captain.Command {
	runner := organizations.NewOrganizations()

	params := organizations.OrgParams{}

	cmd := captain.NewCommand(
		"organizations",
		locale.T("organizations_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			params.Output = globals.Output
			return runner.Run(&params)
		},
	)

	cmd.SetAliases("orgs")

	return cmd
}
