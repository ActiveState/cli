package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/state/internal/commands/state"
)

type CmdTree struct {
	cmd *captain.Command
}

func New() *CmdTree {
	stateCmd := newStateCommand()
	/*{
		activateCmd := newActivateCommand()
		cmd.SetChildren([]*captain.Command{activateCmd})
		{
			activateCmd.SetChildren([]*captain.Command{})
		}
	}*/

	applyLegacyChildren(stateCmd)

	return &CmdTree{
		cmd: stateCmd,
	}
}

type globalOptions struct {
	Verbose bool
}

func newGlobalOptions() *globalOptions {
	return &globalOptions{}
}

func newStateCommand() *captain.Command {
	globals := newGlobalOptions()
	opts := state.NewOptions()

	cmd := state.New(opts)

	return captain.NewCommand(
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
				BoolVar: &globals.Verbose,
			},
			{
				Name:        "version",
				Description: "flag_state_version_description",
				Type:        captain.TypeBool,
				BoolVar:     &opts.Version,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			if globals.Verbose {
				logging.CurrentHandler().SetVerbose(true)
			}

			return cmd.Run(ccmd.Usage)
		},
	)
}

func (ct *CmdTree) Run() error {
	return ct.cmd.Execute()
}
