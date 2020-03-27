package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/deploy"
)

func newDeployCommand(output output.Outputer) *captain.Command {
	runner := deploy.NewDeploy(output)

	params := &deploy.Params{}

	return captain.NewCommand(
		"deploy",
		locale.T("deploy_cmd_description"),
		[]*captain.Flag{
			{
				Name:        "path",
				Description: locale.T("arg_state_deploy_path_description"),
				Value:       &params.Path,
			},
			{
				Name:        "step",
				Description: locale.T("flag_state_deploy_step_description"),
				Value:       &params.Step,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_deploy_namespace"),
				Description: locale.T("arg_state_deploy_namespace_description"),
				Value:       &params.Namespace,
				Required:    true,
			},
		},
		func(cmd *captain.Command, args []string) error {
			return runner.Run(params)
		})
}
