package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/branch"
)

func newBranchCommand(prime *primer.Values) *captain.Command {
	runner := branch.NewList(prime)

	return captain.NewCommand(
		"branch",
		locale.Tl("branch_title", "Listing branches"),
		locale.Tl("branch_description", "Manage your project's branches"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		}).SetGroup(PlatformGroup)
}

func newBranchAddCommand(prime *primer.Values) *captain.Command {
	runner := branch.NewAdd(prime)

	params := branch.AddParams{}

	return captain.NewCommand(
		"add",
		locale.Tl("add_title", "Adding branch"),
		locale.Tl("add_description", "Add a branch to your project"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.Tl("branch_arg_name", "name"),
				Description: locale.Tl("branch_arg_name_description", "Branch to be created"),
				Value:       &params.Label,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		})
}

func newBranchSwitchCommand(prime *primer.Values) *captain.Command {
	runner := branch.NewSwitch(prime)

	params := branch.SwitchParams{}

	return captain.NewCommand(
		"switch",
		locale.Tl("switch_title", "Switching branches"),
		locale.Tl("switch_description", "Switch to the given branch name"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.Tl("branch_switch_arg_name", "switch"),
				Description: locale.Tl("branch_switch_arg_name_description", "Branch to switch to"),
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		})
}
