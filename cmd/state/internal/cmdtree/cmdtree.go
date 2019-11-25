package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/runners/state"
)

type CmdTree struct {
	cmd *captain.Command
}

func New() *CmdTree {
	stateCmd := newStateCommand()
	stateCmd.AddChildren(
		newActivateCommand(),
		newInitCommand(),
		newPushCommand(),
		newRunCommand(),
	)

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

	runner := state.New(opts)
	cmd := captain.NewCommand(
		"state",
		locale.T("state_description"),
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
				Description: locale.T("flag_state_verbose_description"),
				Type:        captain.TypeBool,
				Persist:     true,
				OnUse: func() {
					logging.CurrentHandler().SetVerbose(true)
				},
				BoolVar: &globals.Verbose,
			},
			{
				Name:        "version",
				Description: locale.T("flag_state_version_description"),
				Type:        captain.TypeBool,
				BoolVar:     &opts.Version,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			if globals.Verbose {
				logging.CurrentHandler().SetVerbose(true)
			}

			return runner.Run(ccmd.Usage)
		},
	)

	cmd.SetUsageTemplate("usage_tpl")

	return cmd
}

func (ct *CmdTree) Execute(args []string) error {
	return ct.cmd.Execute(args)
}
