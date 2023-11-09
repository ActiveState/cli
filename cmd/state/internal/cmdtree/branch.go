package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/branch"
	"github.com/ActiveState/cli/internal/runners/swtch"
)

func newBranchCommand(prime *primer.Values) *captain.Command {
	runner := branch.NewList(prime)

	return captain.NewCommand(
		"branch",
		locale.Tl("branch_title", "Listing branches"),
		locale.Tl("branch_description", "Manage your project's branches"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		}).SetGroup(PlatformGroup).SetSupportsStructuredOutput().SetUnstable(true)
}

func newBranchAddCommand(prime *primer.Values) *captain.Command {
	runner := branch.NewAdd(prime)

	params := branch.AddParams{}

	return captain.NewCommand(
		"add",
		locale.Tl("add_title", "Adding branch"),
		locale.Tl("add_description", "Add a branch to your project"),
		prime,
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
		}).SetSupportsStructuredOutput()
}

func newBranchSwitchCommand(prime *primer.Values) *captain.Command {
	runner := swtch.New(prime)

	params := swtch.SwitchParams{}

	cmd := captain.NewCommand(
		"switch",
		locale.Tl("switch_title", "Switching branches"),
		locale.Tl("switch_description", "Switch to the given branch name"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.Tl("switch_arg_identifier", "identifier"),
				Description: locale.Tl("switch_arg_identifier_description", "The commit or branch to switch to"),
				Value:       &params.Identifier,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		})
	cmd.SetSupportsStructuredOutput()
	// We set this command to hidden for backwards compatibility as we cannot
	// alias `state switch` to `state branch switch`
	cmd.SetHidden(true)

	return cmd
}
