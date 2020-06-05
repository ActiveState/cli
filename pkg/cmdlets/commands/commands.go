package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

// T links to locale.T
var T = locale.T

// Tt links to locale.Tt
var Tt = locale.Tt

// Tr links to locale.Tr
var Tr = locale.Tr

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

// Output represents the output type of a command
type Output string

const (
	// JSON is the output type that represents JSON output
	JSON Output = "json"
	// EditorV0 is the output type that represents Komodo output
	EditorV0 Output = "editor.v0"
	// EditorV0 is the output type that represents Editor output
	Editor Output = "editor"
)

// Flag is used to define flags in our Command struct
type Flag struct {
	Name        string
	Shorthand   string
	Description string
	Type        int
	Persist     bool

	OnUse func()

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
	Name               string
	Description        string
	Run                func(cmd *cobra.Command, args []string)
	PersistentPreRun   func(cmd *cobra.Command, args []string)
	Aliases            []string
	Flags              []*Flag
	Arguments          []*Argument
	DisableFlagParsing bool
	Exiter             func(int)
	Hidden             bool

	UsageTemplate string

	cobraCmd   *cobra.Command
	parentCmds []*Command
}

// GetCobraCmd returns the cobra.Command that this struct is wrapping
func (c *Command) GetCobraCmd() *cobra.Command {
	c.Register()
	return c.cobraCmd
}

// Execute the command
func (c *Command) Execute() error {
	failures.ResetHandled()

	c.Register()
	err := c.cobraCmd.Execute()

	fail := failures.Handled()
	if err != nil {
		logging.Error("Error occurred while executing command: %v", err)
	} else if fail != nil {
		logging.Error("Failure occurred while executing command: %v", fail)
		err = fail
	}
	if err != nil {
		c.Exiter(1)
	}

	return err
}

// FlagByName returns the relevant Flag, bubbling up to the parent commands to check for persistent flags
func (c *Command) FlagByName(name string, persistOnly bool) *Flag {
	for _, flag := range c.Flags {
		if flag.Name == name && (!persistOnly || flag.Persist) {
			return flag
		}
	}
	for _, parent := range c.parentCmds {
		if flag := parent.FlagByName(name, true); flag != nil {
			return flag
		}
	}
	return nil
}

// runner wraps the Run command
func (c *Command) runner(cmd *cobra.Command, args []string) {
	outputFlag := cmd.Flag("output")
	if outputFlag != nil && outputFlag.Changed {
		analytics.CustomDimensions.SetOutput(outputFlag.Value.String())
	}
	analytics.Event(analytics.CatRunCmd, c.Name)

	// Run OnUse functions for flags
	if !c.DisableFlagParsing {
		cmd.Flags().VisitAll(func(cobraFlag *pflag.Flag) {
			if !cobraFlag.Changed {
				return
			}

			flag := c.FlagByName(cobraFlag.Name, false)
			if flag == nil || flag.OnUse == nil {
				return
			}

			flag.OnUse()
		})
	}

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

			errMsg += Tr("error_missing_arg", T(arg.Name)) + "\n"
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

// Unregister will essentially forget about the Cobra Command object so that a subsequent call to Register
// will allow for a new Cobra Command and state to be reset.
func (c *Command) Unregister() {
	c.cobraCmd = nil
}

// Register will ensure that we have a cobra.Command registered, if it has already been registered this will do nothing
func (c *Command) Register() {
	if c.cobraCmd != nil {
		return
	}

	if c.Exiter == nil {
		if !condition.InTest() {
			c.Exiter = os.Exit
		} else {
			c.Exiter = func(code int) {
				panic(fmt.Sprintf("Test exited with code %d, you probably want to use testhelpers/exiter.", code))
			}
		}
	}

	c.cobraCmd = &cobra.Command{
		Use:                c.Name,
		Aliases:            c.Aliases,
		Short:              T(c.Description),
		Run:                c.runner,
		PersistentPreRun:   c.PersistentPreRun,
		Args:               c.argInputValidator,
		DisableFlagParsing: c.DisableFlagParsing,
		Hidden:             c.Hidden,
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

	subCmd.parentCmds = append(subCmd.parentCmds, c)
	c.cobraCmd.AddCommand(subCmd.cobraCmd)
}
