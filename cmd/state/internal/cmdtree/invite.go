package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/invite"
)

func newInviteCommand(prime *primer.Values) *captain.Command {
	inviteRunner := invite.New(prime)

	params := invite.Params{}

	return captain.NewCommand(
		"invite",
		locale.Tl("invite_title", "Inviting New Members"),
		locale.Tl("invite_description", "Invite new members to an organization"),
		prime,
		[]*captain.Flag{
			{
				Name:        "organization",
				Description: locale.Tl("invite_flag_organization_description", "Organization to invite to. If not set, invite to current project's organization"),
				Value:       &params.Org,
			},
			{
				Name:        "role",
				Description: locale.Tl("invite_flag_role_description", "Set user role to 'member' or 'owner'. If not set, prompt for the role"),
				Value:       &params.Role,
			},
		},
		[]*captain.Argument{
			{
				Name:        "email1,[email2,..]",
				Description: locale.Tl("invite_arg_emails", "Email addresses to send the invitations to"),
				Required:    true,
				Value:       &params.EmailList,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			return inviteRunner.Run(&params)
		},
	).SetGroup(PlatformGroup).SetUnstable(true)
}
