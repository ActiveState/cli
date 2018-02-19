package commands

import (
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/cobra"
)

// T links to locale.T
var T = locale.T

// Tt links to locale.Tt
var Tt = locale.Tt

// Note we only support the types that we currently have need for. You can add more as needed. Check the pflag docs
// for reference: https://godoc.org/github.com/spf13/pflag
const (
	// TypeString is used to define the type for flags/args
	TypeString = iota
	// TypeInt is used to define the type for flags/args
	TypeInt
)

// Flag is used to define flags in our Command struct
type Flag struct {
	Name        string
	Shorthand   string
	Description string
	Type        int
	Persist     bool

	StringVar   *string
	StringValue string
	IntVar      *int
	IntValue    int
}

// Command covers our command structure, all our commands instantiate a version of this struct
type Command struct {
	Name        string
	Description string
	Run         func(cmd *cobra.Command, args []string)

	Flags []*Flag

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

	for _, flag := range c.Flags {
		c.AddFlag(flag)
	}

	if c.UsageTemplate != "" {
		c.cobraCmd.SetUsageTemplate(Tt(c.UsageTemplate))
	}
}

// AddFlag adds the given flag to our command
func (c *Command) AddFlag(flag *Flag) {
	cc := c.GetCobraCmd()
	flagSetter := cc.Flags
	if flag.Persist {
		flagSetter = cc.PersistentFlags
	}

	switch flag.Type {
	case TypeString:
		flagSetter().StringVarP(flag.StringVar, flag.Name, flag.Shorthand, flag.StringValue, T(flag.Description))
	default:
		logging.Error("Unknown type:" + string(flag.Type))
	}
}

// Append a sub-command to this command
func (c *Command) Append(subCmd *Command) {
	c.Register()
	subCmd.Register()

	c.cobraCmd.AddCommand(subCmd.cobraCmd)
}
