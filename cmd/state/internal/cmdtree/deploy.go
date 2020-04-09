package cmdtree

import (
	"runtime"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/deploy"
)

func newDeployCommand(output output.Outputer) *captain.Command {
	runner := deploy.NewDeploy(output)

	params := &deploy.Params{}

	stepSuffix := ""
	if runtime.GOOS == "linux" {
		stepSuffix = "_linux"
	}

	flags := []*captain.Flag{
		{
			Name:        "path",
			Description: locale.T("flag_state_deploy_path_description"),
			Value:       &params.Path,
		},
		{
			Name:        "step",
			Description: locale.T("flag_state_deploy_step_description" + stepSuffix),
			Value:       &params.Step,
		},
	}

	if runtime.GOOS == "linux" {
		flags = append(flags, &captain.Flag{
			Name:        "force",
			Description: locale.T("flag_state_deploy_force_description"),
			Value:       &params.Force,
		})
	}

	return captain.NewCommand(
		"deploy",
		locale.T("deploy_cmd_description"),
		flags,
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
