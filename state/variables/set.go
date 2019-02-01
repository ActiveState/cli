package variables

import (
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/organizations"
	"github.com/ActiveState/cli/internal/projects"
	"github.com/ActiveState/cli/internal/secrets"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/cobra"
)

func buildSetCommand(cmd *Command) *commands.Command {
	return &commands.Command{
		Name:        "set",
		Description: "variables_set_cmd_description",
		Run:         cmd.ExecuteSet,

		Flags: []*commands.Flag{
			&commands.Flag{
				Name:        "project",
				Shorthand:   "p",
				Description: "variables_set_flag_project",
				Type:        commands.TypeBool,
				BoolVar:     &cmd.Flags.IsProject,
			},
			&commands.Flag{
				Name:        "user",
				Shorthand:   "u",
				Description: "variables_set_flag_user",
				Type:        commands.TypeBool,
				BoolVar:     &cmd.Flags.IsUser,
			},
		},

		Arguments: []*commands.Argument{
			&commands.Argument{
				Name:        "variables_set_arg_name_name",
				Description: "variables_set_arg_name_description",
				Variable:    &cmd.Args.SecretName,
				Required:    true,
			},
			&commands.Argument{
				Name:        "variables_set_arg_value_name",
				Description: "variables_set_arg_value_description",
				Variable:    &cmd.Args.SecretValue,
				Required:    true,
			},
		},
	}
}

// ExecuteSet processes the `secrets set` command.
func (cmd *Command) ExecuteSet(_ *cobra.Command, args []string) {
	currentProject := project.Get()
	org, failure := organizations.FetchByURLName(currentProject.Owner())
	if failure == nil {
		var project *models.Project
		var kp keypairs.Keypair
		if cmd.Flags.IsProject {
			project, failure = projects.FetchByName(org.Urlname, currentProject.Name())
		}

		if failure == nil {
			kp, failure = secrets.LoadKeypairFromConfigDir()
		}

		if failure == nil {
			failure = secrets.Save(cmd.secretsClient, kp, org, project, cmd.Flags.IsUser, cmd.Args.SecretName, cmd.Args.SecretValue)
		}

		if failure == nil && !cmd.Flags.IsUser {
			failure = secrets.ShareWithOrgUsers(cmd.secretsClient, org, project, cmd.Args.SecretName, cmd.Args.SecretValue)
		}
	}

	if failure != nil {
		failures.Handle(failure, locale.T("variables_err"))
	}
}
