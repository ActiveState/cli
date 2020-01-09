package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/runners/state"
	"github.com/ActiveState/cli/state/fork"
	"github.com/ActiveState/cli/state/organizations"
	"github.com/ActiveState/cli/state/pull"
	"github.com/ActiveState/cli/state/scripts"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/ActiveState/cli/state/show"
)

type CmdTree struct {
	cmd *captain.Command
}

func New() *CmdTree {
	globals := newGlobalOptions()

	authCmd := newAuthCommand(globals)
	authCmd.AddChildren(
		newSignupCommand(),
		newLogoutCommand(),
	)

	stateCmd := newStateCommand(globals)
	stateCmd.AddChildren(
		newActivateCommand(globals),
		newInitCommand(),
		newPushCommand(),
		authCmd,
	)

	applyLegacyChildren(stateCmd, globals)

	return &CmdTree{
		cmd: stateCmd,
	}
}

type globalOptions struct {
	Verbose bool
	Output  string
}

func newGlobalOptions() *globalOptions {
	return &globalOptions{}
}

func newStateCommand(globals *globalOptions) *captain.Command {
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
					if !condition.InTest() {
						logging.CurrentHandler().SetVerbose(true)
					}
				},
				BoolVar: &globals.Verbose,
			},
			{
				Name:        "output",
				Shorthand:   "o",
				Description: locale.T("flag_state_output_description"),
				Type:        captain.TypeString,
				Persist:     true,
				StringVar:   &globals.Output,
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

func setLegacyOutput(globals *globalOptions) {
	organizations.Flags.Output = &globals.Output
	scripts.Flags.Output = &globals.Output
	secrets.Flags.Output = &globals.Output
	fork.Flags.Output = &globals.Output
	show.Flags.Output = &globals.Output
	pull.Flags.Output = &globals.Output
}
