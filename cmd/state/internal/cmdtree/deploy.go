package cmdtree

import (
	"runtime"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/deploy"
)

func newDeployCommand(output output.Outputer) *captain.Command {
	runner := deploy.NewDeploy(deploy.UnsetStep, output)

	params := &deploy.Params{}

	flags := []*captain.Flag{
		{
			Name:        "path",
			Description: locale.T("flag_state_deploy_path_description"),
			Value:       &params.Path,
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

func newDeployInstallCommand(output output.Outputer) *captain.Command {
	runner := deploy.NewDeploy(deploy.InstallStep, output)

	params := &deploy.Params{}

	return captain.NewCommand(
		"install",
		locale.T("deploy_install_cmd_description"),
		[]*captain.Flag{
			{
				Name:        "path",
				Description: locale.T("flag_state_deploy_path_description"),
				Value:       &params.Path,
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

func newDeployConfigureCommand(output output.Outputer) *captain.Command {
	runner := deploy.NewDeploy(deploy.ConfigureStep, output)

	params := &deploy.Params{}

	return captain.NewCommand(
		"configure",
		locale.T("deploy_configure_cmd_description"),
		[]*captain.Flag{
			{
				Name:        "path",
				Description: locale.T("flag_state_deploy_path_description"),
				Value:       &params.Path,
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

func newDeploySymlinkCommand(output output.Outputer) *captain.Command {
	runner := deploy.NewDeploy(deploy.SymlinkStep, output)

	params := &deploy.Params{}

	flags := []*captain.Flag{
		{
			Name:        "path",
			Description: locale.T("flag_state_deploy_path_description"),
			Value:       &params.Path,
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
		"symlink",
		locale.T("deploy_symlink_cmd_description"),
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

func newDeployReportCommand(output output.Outputer) *captain.Command {
	runner := deploy.NewDeploy(deploy.ReportStep, output)

	params := &deploy.Params{}

	return captain.NewCommand(
		"report",
		locale.T("deploy_report_cmd_description"),
		[]*captain.Flag{
			{
				Name:        "path",
				Description: locale.T("flag_state_deploy_path_description"),
				Value:       &params.Path,
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
