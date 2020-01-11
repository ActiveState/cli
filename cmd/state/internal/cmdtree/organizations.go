package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/organizations"
)

func newOrganizationsCommand(globals *globalOptions) *captain.Command {
	runner := organizations.NewOrganizations()

	cmd := captain.NewCommand(
		"organizations",
		locale.T("organizations_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&organizations.OrgParams{Output: globals.Output})
		},
	)

	cmd.SetAliases([]string{"orgs"})

	return cmd
}
