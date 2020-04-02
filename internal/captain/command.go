package captain

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type cobraCommander interface {
	GetCobraCmd() *cobra.Command
}

type Executor func(cmd *Command, args []string) error

type Command struct {
	cobra *cobra.Command

	name string

	flags     []*Flag
	arguments []*Argument

	execute func(cmd *Command, args []string) error
}

func NewCommand(name, description string, flags []*Flag, args []*Argument, executor Executor) *Command {
	// Validate args
	for idx, arg := range args {
		if idx > 0 && arg.Required && !args[idx-1].Required {
			msg := fmt.Sprintf(
				"Cannot have a non-required argument followed by a required argument.\n\n%v\n\n%v",
				arg, args[len(args)-1],
			)
			panic(msg)
		}
	}

	cmd := &Command{
		execute:   executor,
		arguments: args,
		flags:     flags,
	}

	cmd.cobra = &cobra.Command{
		Use:              name,
		Short:            description,
		PersistentPreRun: cmd.persistRunner,
		RunE:             cmd.runner,

		// Silence errors and usage, we handle that ourselves
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	if err := cmd.setFlags(flags); err != nil {
		panic(err)
	}
	cmd.SetUsageTemplate("usage_tpl")

	return cmd
}

func (c *Command) Usage() error {
	return c.cobra.Usage()
}

func (c *Command) UsageText() string {
	return c.cobra.UsageString()
}

func (c *Command) Execute(args []string) error {
	c.cobra.SetArgs(args)
	err := c.cobra.Execute()
	c.cobra.SetArgs(nil)
	return setupSensibleErrors(err)
}

func (c *Command) SetAliases(aliases []string) {
	c.cobra.Aliases = aliases
}

func (c *Command) SetDescription(description string) {
	c.cobra.Short = description
}

func (c *Command) SetUsageTemplate(usageTemplate string) {
	localizedArgs := []map[string]string{}
	for _, arg := range c.Arguments() {
		req := ""
		if arg.Required {
			req = "1"
		}
		localizedArgs = append(localizedArgs, map[string]string{
			"Name":        locale.T(arg.Name),
			"Description": locale.T(arg.Description),
			"Required":    req,
		})
	}
	c.cobra.SetUsageTemplate(locale.Tt(usageTemplate, map[string]interface{}{
		"Arguments": localizedArgs,
	}))
}

func (c *Command) SetDisableFlagParsing(b bool) {
	c.cobra.DisableFlagParsing = b
}

func (c *Command) Arguments() []*Argument {
	return c.arguments
}

func (c *Command) AddChildren(children ...*Command) {
	for _, child := range children {
		c.cobra.AddCommand(child.cobra)
	}
}

func (c *Command) AddLegacyChildren(children ...cobraCommander) {
	for _, child := range children {
		c.cobra.AddCommand(child.GetCobraCmd())
	}
}

func (c *Command) flagByName(name string, persistOnly bool) *Flag {
	for _, flag := range c.flags {
		if flag.Name == name && (!persistOnly || flag.Persist) {
			return flag
		}
	}
	return nil
}

func (c *Command) markFlagHidden(name string) error {
	return c.cobra.Flags().MarkHidden(name)
}

func (c *Command) persistRunner(cobraCmd *cobra.Command, args []string) {
	// Run OnUse functions for persistent flags
	c.runFlags(true)
}

func (c *Command) runner(cobraCmd *cobra.Command, args []string) error {
	outputFlag := cobraCmd.Flag("output")
	if outputFlag != nil && outputFlag.Changed {
		analytics.CustomDimensions.SetOutput(outputFlag.Value.String())
	}
	analytics.Event(analytics.CatRunCmd, c.cobra.Name())

	// Run OnUse functions for non-persistent flags
	c.runFlags(false)

	for idx, arg := range c.arguments {
		if idx >= len(args) {
			break
		}

		switch v := arg.Value.(type) {
		case *string:
			*v = args[idx]
		case ArgMarshaler:
			if err := v.Set(args[idx]); err != nil {
				return err
			}
		default:
			return failures.FailDeveloper.New(
				"arg value must be *string, or ArgMarshaler",
			)
		}
	}

	return c.execute(c, args)
}

func (c *Command) runFlags(persistOnly bool) {
	if c.cobra.DisableFlagParsing {
		return
	}

	c.cobra.Flags().VisitAll(func(cobraFlag *pflag.Flag) {
		if !cobraFlag.Changed {
			return
		}

		flag := c.flagByName(cobraFlag.Name, persistOnly)
		if flag == nil || flag.OnUse == nil {
			return
		}

		flag.OnUse()
	})

}

func (c *Command) argValidator(cobraCmd *cobra.Command, args []string) error {
	return nil
}

// setupSensibleErrors inspects an error value for certain errors and returns a
// wrapped error that can be checked and that is localized.
func setupSensibleErrors(err error) error {
	if fail, ok := err.(*failures.Failure); ok && fail == nil {
		return nil
	}
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// pflag: flag.go: output being parsed:
	// fmt.Errorf("invalid argument %q for %q flag: %v", value, flagName, err)
	invalidArg := "invalid argument "
	if strings.Contains(errMsg, invalidArg) {
		segments := strings.SplitN(errMsg, ": ", 2)

		flagText := "{unknown flag}"
		msg := "unknown error"

		if len(segments) > 0 {
			subsegs := strings.SplitN(segments[0], "for ", 2)
			if len(subsegs) > 1 {
				flagText = strings.TrimSuffix(subsegs[1], " flag")
			}
		}

		if len(segments) > 1 {
			msg = segments[1]
		}

		return failures.FailUserInput.New(
			"command_flag_invalid_value", flagText, msg,
		)
	}

	// pflag: flag.go: output being parsed:
	// fmt.Errorf("no such flag -%v", name)
	noSuch := "no such flag "
	if strings.Contains(errMsg, noSuch) {
		flagText := strings.TrimPrefix(errMsg, noSuch)
		return failures.FailUserInput.New(
			"command_flag_no_such_flag", flagText,
		)
	}

	return err
}
