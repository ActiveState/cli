package cmdtree

import (
	"runtime"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/deploy"
	"github.com/ActiveState/cli/internal/runners/deploy/uninstall"
)

func newDeployCommand(prime *primer.Values) *captain.Command {
	runner := deploy.NewDeploy(deploy.UnsetStep, prime)

	params := &deploy.Params{}

	flags := []*captain.Flag{
		{
			Name:        "path",
			Description: locale.T("flag_state_deploy_path_description"),
			Value:       &params.Path,
		},
		{
			Name:        "force",
			Description: locale.T("flag_state_deploy_force_description"),
			Value:       &params.Force,
		},
	}
	if runtime.GOOS == "windows" {
		flags = append(flags, &captain.Flag{
			Name:        "user",
			Description: locale.T("flag_state_deploy_user_path_description"),
			Value:       &params.UserScope,
		})
	}

	cmd := captain.NewCommand(
		"deploy",
		locale.Tl("deploy_title", "Deploying Runtime"),
		locale.T("deploy_cmd_description"),
		prime,
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
	cmd.SetGroup(EnvironmentSetupGroup)
	cmd.SetHidden(true)
	return cmd
}

func newDeployInstallCommand(prime *primer.Values) *captain.Command {
	runner := deploy.NewDeploy(deploy.InstallStep, prime)

	params := &deploy.Params{}

	return captain.NewCommand(
		"install",
		locale.Tl("deploy_install_title", "Installing Runtime (Unconfigured)"),
		locale.T("deploy_install_cmd_description"),
		prime,
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

func newDeployConfigureCommand(prime *primer.Values) *captain.Command {
	runner := deploy.NewDeploy(deploy.ConfigureStep, prime)

	params := &deploy.Params{}

	flags := []*captain.Flag{
		{
			Name:        "path",
			Description: locale.T("flag_state_deploy_path_description"),
			Value:       &params.Path,
		},
	}
	if runtime.GOOS == "windows" {
		flags = append(flags, &captain.Flag{
			Name:        "user",
			Description: locale.T("flag_state_deploy_user_path_description"),
			Value:       &params.UserScope,
		})
	}

	return captain.NewCommand(
		"configure",
		locale.Tl("deploy_configure_title", "Configuring Runtime For Your Shell"),
		locale.T("deploy_configure_cmd_description"),
		prime,
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

func newDeploySymlinkCommand(prime *primer.Values) *captain.Command {
	runner := deploy.NewDeploy(deploy.SymlinkStep, prime)

	params := &deploy.Params{}

	return captain.NewCommand(
		"symlink",
		locale.Tl("deploy_symlink_title", "Symlinking Executables"),
		locale.T("deploy_symlink_cmd_description"),
		prime,
		[]*captain.Flag{
			{
				Name:        "path",
				Description: locale.T("flag_state_deploy_path_description"),
				Value:       &params.Path,
			},
			{
				Name:        "force",
				Description: locale.T("flag_state_deploy_force_description"),
				Value:       &params.Force,
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

func newDeployReportCommand(prime *primer.Values) *captain.Command {
	runner := deploy.NewDeploy(deploy.ReportStep, prime)

	params := &deploy.Params{}

	return captain.NewCommand(
		"report",
		locale.Tl("deploy_report_title", "Reporting Deployment Information"),
		locale.T("deploy_report_cmd_description"),
		prime,
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

func newDeployUninstallCommand(prime *primer.Values) *captain.Command {
	runner := uninstall.NewDeployUninstall(prime)

	params := &uninstall.Params{}

	flags := []*captain.Flag{
		{
			Name:        "path",
			Description: locale.Tl("flag_state_deploy_uninstall_path_description", "The path of the deployed runtime to uninstall if not the current directory"),
			Value:       &params.Path,
		},
	}
	if runtime.GOOS == "windows" {
		flags = append(flags, &captain.Flag{
			Name:        "user",
			Description: locale.T("flag_state_deploy_user_path_description"),
			Value:       &params.UserScope,
		})
	}

	return captain.NewCommand(
		"uninstall",
		locale.Tl("deploy_uninstall_title", "Uninstall Deployed Runtime"),
		locale.Tl("deploy_uninstall_cmd_description", "Removes a runtime that had previously been deployed"),
		prime,
		flags,
		[]*captain.Argument{},
		func(cmd *captain.Command, args []string) error {
			return runner.Run(params)
		})
}
