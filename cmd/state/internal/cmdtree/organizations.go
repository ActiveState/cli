package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/organizations"
)

func newOrganizationsCommand(prime *primer.Values) *captain.Command {
	runner := organizations.NewOrganizations(prime)

	params := organizations.OrgParams{}

	cmd := captain.NewCommand(
		"organizations",
		locale.T("organizations_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&params)
		},
	)

	cmd.SetAliases("orgs")

	return cmd
}
