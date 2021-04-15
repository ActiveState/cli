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
	"github.com/rollbar/rollbar-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/output/txtstyle"
	"github.com/ActiveState/cli/internal/sighandler"
	"github.com/ActiveState/cli/internal/table"
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
	parent   *Command

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
	cfg analytics.Configurable
}

func NewCommand(name, title, description string, out output.Outputer, cfg analytics.Configurable, flags []*Flag, args []*Argument, execute ExecuteFunc) *Command {
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
		cfg:       cfg,
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
		panic(errs.Join(err, "\n").Error())
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
func NewHiddenShimCommand(name string, cfg analytics.Configurable, flags []*Flag, args []*Argument, execute ExecuteFunc) *Command {
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
		cfg:       cfg,
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
func NewShimCommand(name, description string, cfg analytics.Configurable, execute ExecuteFunc) *Command {
	cmd := &Command{
		execute: execute,
		cfg:     cfg,
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

func (c *Command) Hidden() bool {
	return c.cobra.Hidden
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

func (c *Command) NameRecursive() string {
	child := c
	name := []string{}
	for child != nil {
		name = append([]string{child.Name()}, name...)
		child = child.parent
	}
	return strings.Join(name, " ")
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

		if child.parent != nil {
			panic(fmt.Sprintf("Command %s already has a parent: %s", child.Name(), child.parent.Name()))
		}
		child.parent = c

		interceptChain := append(c.interceptChain, child.interceptChain...)
		child.SetInterceptChain(interceptChain...)
	}
}

func (c *Command) AddLegacyChildren(children ...*cobra.Command) {
	for _, child := range children {
		c.cobra.AddCommand(child)
	}
}

func (c *Command) topLevelCobra() *cobra.Command {
	parent := c.cobra
	for parent.HasParent() {
		parent = parent.Parent()
	}
	return parent
}

func (c *Command) Parent() *Command {
	return c.parent
}

func (c *Command) TopParent() *Command {
	child := c
	for {
		if child.parent == nil {
			return child
		}
		child = child.parent
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

func (c *Command) GenBashCompletions() (string, error) {
	buf := new(bytes.Buffer)
	if err := c.topLevelCobra().GenBashCompletion(buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (c *Command) GenFishCompletions() (string, error) {
	buf := new(bytes.Buffer)
	if err := c.topLevelCobra().GenFishCompletion(buf, true); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (c *Command) GenPowerShellCompletion() (string, error) {
	buf := new(bytes.Buffer)
	if err := c.topLevelCobra().GenPowerShellCompletion(buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (c *Command) GenZshCompletion() (string, error) {
	buf := new(bytes.Buffer)
	if err := c.topLevelCobra().GenZshCompletion(buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (c *Command) flagByName(name string, persistOnly bool) *Flag {
	for _, flag := range c.flags {
		if flag.Name == name && (!persistOnly || flag.Persist) {
			return flag
		}
	}
	return nil
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
	analytics.SetDeferred(c.cfg, c.deferAnalytics)

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
			return locale.NewInputError("err_arg_required", "", arg.Name, arg.Description)
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
			return locale.NewError("err_arg_invalid_type", "arg: {{.V0}} must be *string, or ArgMarshaler", arg.Name)
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

	exitCode := errs.UnwrapExitCode(err)

	var serr interface{ Signal() os.Signal }
	if errors.As(err, &serr) {
		analytics.EventWithLabel(analytics.CatCommandExit, subCommandString, "interrupt")
		err = locale.WrapInputError(err, "user_interrupt", "User interrupted the State Tool process.")
	} else {
		analytics.EventWithLabel(analytics.CatCommandExit, subCommandString, strconv.Itoa(exitCode))
	}
	events.WaitForEvents(1*time.Second, analytics.Wait, rollbar.Wait)

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
	if err, ok := err.(error); ok && err == nil {
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

		return locale.NewInputError("command_flag_invalid_value", "", flagText, msg)
	}

	if pflagErrFlag := pflagFlagErrMsgFlag(errMsg); pflagErrFlag != "" {
		return locale.NewInputError(
			"command_flag_no_such_flag",
			"No such flag: [NOTICE]{{.V0}}[/RESET]", pflagErrFlag,
		)
	}

	if pflagErrCmd := pflagCmdErrMsgCmd(errMsg); pflagErrCmd != "" {
		return locale.NewInputError(
			"command_cmd_no_such_cmd",
			"No such command: [NOTICE]{{.V0}}[/RESET]", pflagErrCmd,
		)
	}

	return err
}

// pflag: flag.go: errors are not detectable
func pflagFlagErrMsgFlag(errMsg string) string {
	flagText := strings.TrimPrefix(errMsg, "no such flag ")
	flagText = strings.TrimPrefix(flagText, "unknown flag: ")
	flagText = strings.TrimPrefix(flagText, "bad flag syntax: ")
	//unknown shorthand flag: 'x' in -x
	flagText = strings.TrimPrefix(flagText, "unknown shorthand flag: ")

	if flagText == errMsg {
		return ""
	}

	shorthandSplit := strings.Split(flagText, "' in ")
	if len(shorthandSplit) > 1 {
		flagText = shorthandSplit[1]
	}

	return flagText
}

func pflagCmdErrMsgCmd(errMsg string) string {
	// unknown command "badcmd" for "state"
	flagText := strings.TrimPrefix(errMsg, `unknown command "`)

	if flagText == errMsg {
		return ""
	}

	commandSplit := strings.Split(flagText, `" for "`)
	if len(commandSplit) > 1 && commandSplit[0] != "" {
		flagText = commandSplit[0]
	}

	return flagText
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
		"childCommands": childCommands,
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

func childCommands(cmd *Command) string {
	if len(cmd.AvailableChildren()) == 0 {
		return ""
	}

	var group string
	table := table.New([]string{"", ""})
	table.HideHeaders = true
	for _, child := range cmd.Children() {
		if group != child.Group().String() && child.Group().String() != "" {
			group = child.Group().String()
			table.AddRow([]string{""})
			table.AddRow([]string{fmt.Sprintf("%s:", group)})
		}
		if !child.cobra.Hidden {
			table.AddRow([]string{fmt.Sprintf("  %s", child.Name()), child.ShortDescription()})
		}
	}

	return fmt.Sprintf("Available Commands:\n%s", table.Render())
}
