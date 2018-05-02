package commands

import (
	"fmt"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/spf13/cobra"
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
	// TypeBool is used to define the type for flags/args
	TypeBool
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
	BoolVar     *bool
	BoolValue   bool
}

// Argument is used to define flags in our Command struct
type Argument struct {
	Name        string
	Description string
	Required    bool
	Validator   func(arg *Argument, value string) error
	Variable    *string
}

// Command covers our command structure, all our commands instantiate a version of this struct
type Command struct {
	Name        string
	Description string
	Run         func(cmd *cobra.Command, args []string)

	Flags     []*Flag
	Arguments []*Argument

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
	analytics.Event(analytics.CatRunCmd, c.Name)
	for idx, arg := range c.Arguments {
		if len(args) > idx {
			(*arg.Variable) = args[idx]
		}
	}
	c.Run(cmd, args)
}

// argInputValidator validates whether we have all the args we need
func (c *Command) argInputValidator(cmd *cobra.Command, args []string) error {

	// Validate whether we have all arguments that need to be defined
	if len(args) < len(c.Arguments) {
		errMsg := ""
		for i := len(args); i < len(c.Arguments); i++ {
			arg := c.Arguments[i]
			if !arg.Required {
				break
			}

			errMsg += T("error_missing_arg", c.Arguments[i]) + "\n"
		}
		if errMsg != "" {
			return failures.FailUserInput.New(errMsg)
		}
	}

	size := len(args)
	if len(c.Arguments) < size {
		size = len(c.Arguments)
	}

	// Invoke validators, if any are defined
	errMsg := ""
	for i := 0; i < size; i++ {
		arg := c.Arguments[i]
		if arg.Validator == nil {
			continue
		}

		err := arg.Validator(arg, args[i])
		if err != nil {
			errMsg += err.Error() + "\n"
		}
	}

	if errMsg != "" {
		return failures.FailUserInput.New(errMsg)
	}

	return nil
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
		Args:  c.argInputValidator,
	}

	for _, flag := range c.Flags {
		err := c.AddFlag(flag)
		if err != nil {
			// This only happens if our code is plain wrong, so the panic is a quality of life "feature" for us devs
			panic(err.Error())
		}
	}

	for idx, arg := range c.Arguments {
		err := c.validateAddArgument(arg, idx)
		if err != nil {
			// This only happens if our code is plain wrong, so the panic is a quality of life "feature" for us devs
			panic(err.Error())
		}
	}

	if c.UsageTemplate == "" {
		c.UsageTemplate = "usage_tpl"
	}

	args := []map[string]string{}
	for _, arg := range c.Arguments {
		req := ""
		if arg.Required {
			req = "1"
		}
		args = append(args, map[string]string{"Name": T(arg.Name), "Description": T(arg.Description), "Required": req})
	}
	c.cobraCmd.SetUsageTemplate(Tt(c.UsageTemplate, map[string]interface{}{
		"Arguments": args,
	}))
}

// AddFlag adds the given flag to our command
func (c *Command) AddFlag(flag *Flag) error {
	cc := c.GetCobraCmd()
	flagSetter := cc.Flags
	if flag.Persist {
		flagSetter = cc.PersistentFlags
	}

	switch flag.Type {
	case TypeString:
		flagSetter().StringVarP(flag.StringVar, flag.Name, flag.Shorthand, flag.StringValue, T(flag.Description))
	case TypeInt:
		flagSetter().IntVarP(flag.IntVar, flag.Name, flag.Shorthand, flag.IntValue, T(flag.Description))
	case TypeBool:
		flagSetter().BoolVarP(flag.BoolVar, flag.Name, flag.Shorthand, flag.BoolValue, T(flag.Description))
	default:
		return failures.FailInput.New("Unknown type:" + string(flag.Type))
	}

	return nil
}

// AddArgument adds the given argument to our command
func (c *Command) validateAddArgument(arg *Argument, idx int) error {
	if idx == -1 {
		idx = len(c.Arguments) - 1
	}
	if idx > 0 && arg.Required && !c.Arguments[idx-1].Required {
		return failures.FailInput.New(
			fmt.Sprintf("Cannot have a non-required argument followed by a required argument.\n\n%v\n\n%v",
				arg, c.Arguments[len(c.Arguments)-1]))
	}
	return nil
}

// AddArgument adds the given argument to our command
func (c *Command) AddArgument(arg *Argument) error {
	err := c.validateAddArgument(arg, -1)
	if err != nil {
		return err
	}
	c.Arguments = append(c.Arguments, arg)
	return nil
}

// Append a sub-command to this command
func (c *Command) Append(subCmd *Command) {
	c.Register()
	subCmd.Register()

	c.cobraCmd.AddCommand(subCmd.cobraCmd)
}
