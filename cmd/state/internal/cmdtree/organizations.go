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
		locale.Tl("organizations_title", "Listing Organizations"),
		locale.T("organizations_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&params)
		},
	)

	cmd.SetGroup(PlatformGroup)
	cmd.SetAliases("orgs")
	cmd.SetUnstable(true)

	return cmd
}

func newOrganizationsAddCommand(prime *primer.Values) *captain.Command {
	runner := organizations.NewOrganizationsAdd(prime)

	params := organizations.OrgAddParams{}

	cmd := captain.NewCommand(
		"add",
		locale.Tl("organizations_add_title", "Creating Organization"),
		locale.T("organizations_add_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.Tl("organizations_add_name", "Name"),
				Description: locale.Tl("organizations_add_name_description", "The name of the organization"),
				Required:    true,
				Value:       &params.Name,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&params)
		},
	)

	cmd.SetUnstable(true)
	cmd.SetHidden(true) // for test use only at this time

	return cmd
}
