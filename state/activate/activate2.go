package activate

import "github.com/ActiveState/cli/pkg/cmdlets/commands"

type ActivateCommand struct {
	meta      commands.Meta
	locale    commands.Locale
	flags     []*commands.Flag
	arguments []*commands.Argument
	options   []commands.Option
}

func NewActivateCommand() commands.Commander {
	return &ActivateCommand{
		meta: commands.Meta{
			Name: "activate",
		},
		locale: commands.Locale{
			Description: "activate_project",
		},
		flags: []*commands.Flag{
			&commands.Flag{
				Name:        "path",
				Shorthand:   "",
				Description: "flag_state_activate_path_description",
				Type:        commands.TypeString,
				StringVar:   &Flags.Path,
			},
		},
		arguments: []*commands.Argument{
			&commands.Argument{
				Name:        "arg_state_activate_namespace",
				Description: "arg_state_activate_namespace_description",
				Variable:    &Args.Namespace,
			},
		},
		options: []commands.Option{
			commands.OptionHidden(),
		},
	}
}

func (cmd *ActivateCommand) Execute() error {
	return nil
}

func (cmd *ActivateCommand) Meta() commands.Meta {
	return cmd.meta
}

func (cmd *ActivateCommand) Locale() commands.Locale {
	return cmd.locale
}

func (cmd *ActivateCommand) Flags() []*commands.Flag {
	return cmd.flags
}

func (cmd *ActivateCommand) Arguments() []*commands.Argument {
	return cmd.arguments
}

func (cmd *ActivateCommand) Options() []commands.Option {
	return cmd.options
}
