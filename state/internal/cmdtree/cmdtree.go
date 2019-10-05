package main

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

type CmdTree struct {
	cmd *captain.Command
}

func NewCmdTree() *CmdTree {
	stateCmd := newStateCommand()
	{
		activateCmd := newActivateCommand()
		stateCmd.SetChildren([]*captain.Command{activateCmd})
		{
			activateCmd.SetChildren([]*captain.Command{})
		}
	}

	return &CmdTree{
		cmd: stateCmd,
	}
}

func newStateCommand() *captain.Command {
	opts := &StateOptions{}
	stateRunner := NewStateRunner(opts)
	stateCmd := captain.NewCommand(
		"state",
		[]*captain.Flag{
			{
				Name:        "locale",
				Shorthand:   "l",
				Description: locale.T("flag_state_locale_description"),
				Type:        captain.TypeString,
				Persist:     true,
				StringVar:   &opts.Locale,
			},
			{
				Name:        "verbose",
				Shorthand:   "v",
				Description: "flag_state_verbose_description",
				Type:        captain.TypeBool,
				Persist:     true,
				OnUse: func() {
					logging.CurrentHandler().SetVerbose(true)
				},
				BoolVar: &opts.Verbose,
			},
			{
				Name:        "version",
				Description: "flag_state_version_description",
				Type:        captain.TypeBool,
				BoolVar:     &opts.Version,
			},
		},
		[]*captain.Argument{},
		func(cmd *captain.Command, args []string) error { return stateRunner.Execute(cmd.Usage) },
	)
	return stateCmd
}

func newActivateCommand() *captain.Command {
	return nil
}

func (ct *CmdTree) Run() error {
	return ct.cmd.Execute()
}
