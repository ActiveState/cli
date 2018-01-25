package structures

import (
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/cobra"
)

var T = locale.T
var Tt = locale.Tt

type Command struct {
	Name        string
	Description string
	Run         func(cmd *cobra.Command, args []string)

	UsageTemplate string

	cobraCmd *cobra.Command
}

// GetCobraCmd returns the cobra.Command that this struct is wrapping
func (c *Command) GetCobraCmd() *cobra.Command {
	c.Register()
	return c.cobraCmd
}

// Execute the command
func (c *Command) Execute() error {
	c.Register()
	return c.cobraCmd.Execute()
}

// runner wraps the Run command
func (c *Command) runner(cmd *cobra.Command, args []string) {
	c.Run(cmd, args)
}

// Register will ensure that we have a cobra.Command registered, if it has already been registered this will do nothing
func (c *Command) Register() {
	if c.cobraCmd != nil {
		return
	}

	c.cobraCmd = &cobra.Command{
		Use:   c.Name,
		Short: T(c.Description),
		Run:   c.runner,
	}

	if c.UsageTemplate != "" {
		c.cobraCmd.SetUsageTemplate(Tt(c.UsageTemplate))
	}
}

// Append a sub-command to this command
func (c *Command) Append(subCmd *Command) {
	c.Register()
	subCmd.Register()

	c.cobraCmd.AddCommand(subCmd.cobraCmd)
}
