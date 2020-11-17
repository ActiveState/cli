package captain

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/gobuffalo/packr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/output/txtstyle"
	"github.com/ActiveState/cli/internal/sighandler"
)

var cobraMapping map[*cobra.Command]*Command = make(map[*cobra.Command]*Command)

type cobraCommander interface {
	GetCobraCmd() *cobra.Command
}

type ExecuteFunc func(cmd *Command, args []string) error

type InterceptFunc func(ExecuteFunc) ExecuteFunc

type CommandGroup struct {
	name     string
	priority int
}

func (c CommandGroup) String() string {
	return c.name
}

func (c CommandGroup) SortBefore(c2 CommandGroup) bool {
	if c.priority != 0 {
		return c.priority > c2.priority
	}
	return c.name < c2.name
}

func NewCommandGroup(name string, priority int) CommandGroup {
	return CommandGroup{name, priority}
}

type Command struct {
	cobra    *cobra.Command
	commands []*Command

	title string

	group CommandGroup

	flags     []*Flag
	arguments []*Argument

	execute        ExecuteFunc
	interceptChain []InterceptFunc

	// deferAnalytics should be set if the command handles the GA reporting in its execute function
	deferAnalytics bool

	skipChecks bool

	out output.Outputer
}

func NewCommand(name, title, description string, out output.Outputer, flags []*Flag, args []*Argument, execute ExecuteFunc) *Command {
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
		title:     title,
		execute:   execute,
		arguments: args,
		flags:     flags,
		commands:  make([]*Command, 0),
		out:       out,
	}

	short := description
	if idx := strings.IndexByte(description, '.'); idx > 0 {
		short = description[0:idx]
	}

	cmd.cobra = &cobra.Command{
		Use:              name,
		Short:            short,
		Long:             description,
		PersistentPreRun: cmd.persistRunner,
		RunE:             cmd.runner,

		// Silence errors and usage, we handle that ourselves
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	if err := cmd.setFlags(flags); err != nil {
		panic(err)
	}

	cmd.cobra.SetUsageFunc(func(c *cobra.Command) error {
		err := cmd.Usage()
		if err != nil {
			// Cobra doesn't return this error for us, so we have to ensure it's logged
			logging.Error("Error while running usage: %v", err)
		}
		return err
	})

	cobraMapping[cmd.cobra] = cmd
	return cmd
}

// NewHiddenShimCommand is a very specialized function that is used for adding the
// PPM Shim.  Differences to NewCommand() are:
// - the entrypoint is hidden in the help text
// - calling the help for a subcommand will execute this subcommand
func NewHiddenShimCommand(name string, flags []*Flag, args []*Argument, execute ExecuteFunc) *Command {
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
		execute:   execute,
		arguments: args,
		flags:     flags,
	}

	cmd.cobra = &cobra.Command{
		Use:              name,
		PersistentPreRun: cmd.persistRunner,
		RunE:             cmd.runner,
		Hidden:           true,

		// Silence errors and usage, we handle that ourselves
		SilenceErrors:      true,
		SilenceUsage:       true,
		DisableFlagParsing: true,
	}

	cmd.cobra.SetHelpFunc(func(_ *cobra.Command, args []string) {
		cmd.execute(cmd, args)
	})

	if err := cmd.setFlags(flags); err != nil {
		panic(err)
	}

	return cmd
}

// NewShimCommand is a very specialized function that is used to support sub-commands for a hidden shim command.
// It has only a name a description and function to execute.  All flags and arguments are ignored.
func NewShimCommand(name, description string, execute ExecuteFunc) *Command {
	cmd := &Command{
		execute: execute,
	}

	short := description
	if idx := strings.IndexByte(description, '.'); idx > 0 {
		short = description[0:idx]
	}

	cmd.cobra = &cobra.Command{
		Use:                name,
		Short:              short,
		Long:               description,
		DisableFlagParsing: true,
		RunE:               cmd.runner,
	}

	return cmd
}

func (c *Command) Use() string {
	return c.cobra.Use
}

func (c *Command) UseFull() string {
	return strings.Join(c.subCommandNames(), " ")
}

func (c *Command) UsageText() string {
	return c.cobra.UsageString()
}

func (c *Command) Help() string {
	return fmt.Sprintf("%s\n\n%s", c.cobra.Short, c.UsageText())
}

func (c *Command) ShortDescription() string {
	return c.cobra.Short
}

func (c *Command) Execute(args []string) error {
	c.cobra.SetArgs(args)
	err := c.cobra.Execute()
	c.cobra.SetArgs(nil)
	return setupSensibleErrors(err)
}

func (c *Command) SetAliases(aliases ...string) {
	c.cobra.Aliases = aliases
}

func (c *Command) SetDeferAnalytics(value bool) {
	c.deferAnalytics = value
}

func (c *Command) SetSkipChecks(value bool) {
	c.skipChecks = value
}

func (c *Command) SetHidden(value bool) {
	c.cobra.Hidden = value
}

func (c *Command) SetDescription(description string) {
	c.cobra.Short = description
}

func (c *Command) SetDisableFlagParsing(b bool) {
	c.cobra.DisableFlagParsing = b
}

func (c *Command) Name() string {
	return c.cobra.Name()
}

func (c *Command) NamePadding() int {
	return c.cobra.NamePadding()
}

func (c *Command) Title() string {
	return c.title
}

func (c *Command) Description() string {
	return c.cobra.Long
}

func (c *Command) Flags() []*Flag {
	return c.flags
}

func (c *Command) ExecuteFunc() ExecuteFunc {
	return c.execute
}

func (c *Command) Arguments() []*Argument {
	return c.arguments
}

func (c *Command) SetInterceptChain(fns ...InterceptFunc) {
	c.interceptChain = fns
}

func (c *Command) interceptFunc() InterceptFunc {
	return func(fn ExecuteFunc) ExecuteFunc {
		for i := len(c.interceptChain) - 1; i >= 0; i-- {
			if c.interceptChain[i] == nil {
				continue
			}
			fn = c.interceptChain[i](fn)
		}
		return fn
	}
}

// SetGroup sets the group this command belongs to. This defaults to empty, meaning the command is ungrouped.
// Realistically only top level commands really need a group.
func (c *Command) SetGroup(group CommandGroup) *Command {
	c.group = group
	return c
}

func (c *Command) Group() CommandGroup {
	return c.group
}

func (c *Command) SkipChecks() bool {
	return c.skipChecks
}

func (c *Command) SortBefore(c2 *Command) bool {
	if c.group != c2.group {
		return c.group.SortBefore(c2.group)
	}
	return c.Name() < c2.Name()
}

func (c *Command) AddChildren(children ...*Command) {
	for _, child := range children {
		c.commands = append(c.commands, child)
		c.cobra.AddCommand(child.cobra)

		interceptChain := append(c.interceptChain, child.interceptChain...)
		child.SetInterceptChain(interceptChain...)
	}
}

func (c *Command) AddLegacyChildren(children ...cobraCommander) {
	for _, child := range children {
		c.cobra.AddCommand(child.GetCobraCmd())
	}
}

func (c *Command) Children() []*Command {
	commands := c.commands
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].SortBefore(commands[j])
	})
	return commands
}

func (c *Command) AvailableChildren() []*Command {
	commands := []*Command{}
	for _, child := range c.Children() {
		if !child.cobra.IsAvailableCommand() {
			continue
		}
		commands = append(commands, child)
	}
	return commands
}

func (c *Command) Find(args []string) (*Command, error) {
	foundCobra, _, err := c.cobra.Find(args)
	if err != nil {
		return nil, errs.Wrap(err, "Could not find child command with args: %s", strings.Join(args, " "))
	}
	if cmd, ok := cobraMapping[foundCobra]; ok {
		return cmd, nil
	}
	return nil, locale.NewError("err_captain_cmd_find", "Could not find child Command with args: {{.V0}}", strings.Join(args, " "))
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

// subCommandNames returns a slice of the names of the sub-commands called
func (c *Command) subCommandNames() []string {
	var commands []string
	cmd := c.cobra
	root := cmd.Root()
	for {
		if cmd == nil || cmd == root {
			break
		}
		commands = append(commands, cmd.Name())
		cmd = cmd.Parent()
	}

	// reverse commands
	for i, j := 0, len(commands)-1; i < j; i, j = i+1, j-1 {
		commands[i], commands[j] = commands[j], commands[i]
	}

	return commands
}

func (c *Command) runner(cobraCmd *cobra.Command, args []string) error {
	analytics.SetDeferred(c.deferAnalytics)

	outputFlag := cobraCmd.Flag("output")
	if outputFlag != nil && outputFlag.Changed {
		analytics.CustomDimensions.SetOutput(outputFlag.Value.String())
	}
	subCommandString := c.UseFull()

	// Send  GA events unless they are handled in the runners...
	analytics.Event(analytics.CatRunCmd, subCommandString)

	// Run OnUse functions for non-persistent flags
	c.runFlags(false)

	for idx, arg := range c.arguments {
		if arg.Required && idx > len(args)-1 {
			return failures.FailUserInput.New(locale.Tr("err_arg_required", arg.Name, arg.Description))
		}

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
				fmt.Sprintf("arg: %s must be *string, or ArgMarshaler", arg.Name),
			)
		}

	}

	if c.out != nil && c.title != "" {
		c.out.Notice(txtstyle.NewTitle(c.title))
	}

	intercept := c.interceptFunc()
	execute := intercept(c.execute)

	// initialize signal handler for analytics events
	as := sighandler.NewAwaitingSigHandler(os.Interrupt)
	sighandler.Push(as)
	defer sighandler.Pop()

	err := as.WaitForFunc(func() error {
		return execute(c, args)
	})

	exitCode := errs.UnwrapExitCode(failures.ToError(err))

	var serr interface{ Signal() os.Signal }
	if errors.As(err, &serr) {
		analytics.EventWithLabel(analytics.CatCommandExit, subCommandString, "interrupt")
		err = locale.WrapInputError(err, "user_interrupt", "User interrupted the State Tool process.")
	} else {
		analytics.EventWithLabel(analytics.CatCommandExit, subCommandString, strconv.Itoa(exitCode))
	}
	analytics.WaitForAllEvents(time.Second * 1)

	return err
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

	if strings.Contains(errMsg, "unknown command") {
		return locale.NewInputError("err_cobra_unknown_cmd", "{{.V0}}", errMsg)
	}

	return err
}

func (cmd *Command) Usage() error {
	tpl := template.New("usage")
	tpl.Funcs(template.FuncMap{
		"rpad": func(s string, padding int) string {
			template := fmt.Sprintf("%%-%ds", padding)
			return fmt.Sprintf(template, s)
		},
		"trimTrailingWhitespaces": func(s string) string {
			return strings.TrimRightFunc(s, unicode.IsSpace)
		},
	})

	box := packr.NewBox("../../assets")

	var err error
	if tpl, err = tpl.Parse(box.String("usage.tpl")); err != nil {
		return errs.Wrap(err, "Could not parse template")
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, map[string]interface{}{
		"Cmd":   cmd,
		"Cobra": cmd.cobra,
	}); err != nil {
		return errs.Wrap(err, "Could not execute template")
	}

	cmd.out.Print(out.String())

	return nil
}
