package activate

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/spf13/cobra"
)

type ActivateCommand struct {
	meta      captain.Meta
	locale    captain.Locale
	flags     []*captain.Flag
	arguments []*captain.Argument
	options   []captain.Option
}

func NewActivateCommand() captain.Commander {
	return &ActivateCommand{
		meta: captain.Meta{
			Name: "activate",
		},
		locale: captain.Locale{
			Description: "activate_project",
		},
		flags: []*captain.Flag{
			&captain.Flag{
				Name:        "path",
				Shorthand:   "",
				Description: "flag_state_activate_path_description",
				Type:        captain.TypeString,
			},
		},
		arguments: []*captain.Argument{
			&captain.Argument{
				Name:        "arg_state_activate_namespace",
				Description: "arg_state_activate_namespace_description",
			},
		},
		options: []captain.Option{
			captain.OptionHidden(),
		},
	}
}

func (c *ActivateCommand) Execute(cmd *cobra.Command, args []string) error {
	return nil
}

func (c *ActivateCommand) Meta() captain.Meta {
	return c.meta
}

func (c *ActivateCommand) Locale() captain.Locale {
	return c.locale
}

func (c *ActivateCommand) Flags() []*captain.Flag {
	return c.flags
}

func (c *ActivateCommand) Arguments() []*captain.Argument {
	return c.arguments
}

func (c *ActivateCommand) Options() []captain.Option {
	return c.options
}

func (c *ActivateCommand) Children() []captain.Commander {
	return []captain.Commander{}
}
