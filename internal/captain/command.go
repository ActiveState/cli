package captain

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Commander interface {
	Execute(cmd *cobra.Command, args []string) error
	Meta() Meta
	Locale() Locale
	Flags() []*Flag
	Arguments() []*Argument
	Options() []Option
	Children() []Commander
}

type Meta struct {
	Name    string
	Aliases []string
}

type Locale struct {
	Description   string
	UsageTemplate string
}

type Platoon struct {
	cmd   Commander
	cobra *cobra.Command
}

func (c *Platoon) Execute() error {
	return c.cobra.Execute()
}

func (c *Platoon) flagByName(name string, persistOnly bool) *Flag {
	for _, flag := range c.cmd.Flags() {
		if flag.Name == name && (!persistOnly || flag.Persist) {
			return flag
		}
	}
	return nil
}

func (c *Platoon) runner(cobraCmd *cobra.Command, args []string) error {
	analytics.Event(analytics.CatRunCmd, c.cmd.Meta().Name)

	// Run OnUse functions for flags
	if !cobraCmd.DisableFlagParsing {
		cobraCmd.Flags().VisitAll(func(cobraFlag *pflag.Flag) {
			if !cobraFlag.Changed {
				return
			}

			flag := c.flagByName(cobraFlag.Name, false)
			if flag == nil || flag.OnUse == nil {
				return
			}

			flag.OnUse()
		})
	}

	for idx, arg := range c.cmd.Arguments() {
		if len(args) > idx {
			(*arg.Variable) = args[idx]
		}
	}
	return c.cmd.Execute(cobraCmd, args)
}

func (c *Platoon) argValidator(cobraCmd *cobra.Command, args []string) error {
	return nil
}
