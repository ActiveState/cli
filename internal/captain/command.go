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

	"github.com/ActiveState/cli/internal/analytics"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/sighandler"
	"github.com/ActiveState/cli/internal/table"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ErrNoChildren struct{ *locale.LocalizedError }

func init() {
	configMediator.RegisterOption(constants.UnstableConfig, configMediator.Bool, false)
}

// appEventPrefix is used for all executables except for the State Tool itself.
var appEventPrefix string = func() string {
	execName := osutils.ExecutableName()
	if execName == constants.CommandName {
		return ""
	}
	return execName + " "
}()

var cobraMapping map[*cobra.Command]*Command = make(map[*cobra.Command]*Command)

type primer interface {
	Output() output.Outputer
	Analytics() analytics.Dispatcher
	Config() *config.Instance
}

type ExecuteFunc func(cmd *Command, args []string) error

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

	name  string
	title string

	group CommandGroup

	deprioritizeInHelpListing bool

	flags     []*Flag
	arguments []*Argument

	execute     ExecuteFunc
	onExecStart []ExecEventHandler
	onExecStop  []ExecEventHandler

	// deferAnalytics should be set if the command handles the GA reporting in its execute function
	deferAnalytics bool

	skipChecks bool

	unstable         bool
	structuredOutput bool

	examples []string

	out       output.Outputer
	analytics analytics.Dispatcher
	cfg       *config.Instance
}

func NewCommand(name, title, description string, prime primer, flags []*Flag, args []*Argument, execute ExecuteFunc) *Command {
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
		name:      name,
		title:     title,
		execute:   execute,
		arguments: args,
		flags:     flags,
		commands:  make([]*Command, 0),
		out:       prime.Output(),
		analytics: prime.Analytics(),
		cfg:       prime.Config(),
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
		RunE:             cmd.cobraExecHandler,

		// Restrict command line arguments by default.
		// cmd.SetHasVariableArguments() overrides this.
		Args: cobra.MaximumNArgs(len(args)),

		// Silence errors and usage, we handle that ourselves
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	if err := cmd.setFlags(flags); err != nil {
		panic(errs.JoinMessage(err))
	}

	cmd.cobra.SetUsageFunc(func(c *cobra.Command) error {
		err := cmd.Usage()
		if err != nil {
			// Cobra doesn't return this error for us, so we have to ensure it's logged
			multilog.Error("Error while running usage: %v", err)
		}
		return err
	})

	// When there are errors processing flags, the command's runner is not called.
	// In addition to performing the unstable feature check and printing of the unstable banner in
	// the command's runner, we need to do it here too if there's an error processing flags.
	// If the command is not unstable, simply return the flag error.
	cmd.cobra.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		if cmd.shouldWarnUnstable() {
			if !condition.OptInUnstable(cmd.cfg) {
				cmd.out.Notice(locale.T("unstable_command_warning"))
				return nil
			}
			cmd.outputTitleIfAny()
		} else if cmd.out.Type().IsStructured() && !cmd.structuredOutput {
			return locale.NewInputError("err_no_structured_output", "", string(cmd.out.Type()))
		}
		return err
	})

	cobraMapping[cmd.cobra] = cmd
	return cmd
}

func (c *Command) Use() string {
	return c.cobra.Use
}

func (c *Command) JoinedSubCommandNames() string {
	return strings.Join(c.commandNames(false), " ")
}

func (c *Command) JoinedCommandNames() string {
	return strings.Join(c.commandNames(true), " ")
}

func (c *Command) UsageText() string {
	return c.cobra.UsageString()
}

func (c *Command) Help() string {
	return strings.TrimRightFunc(fmt.Sprintf("%s\n\n%s", c.cobra.Short, c.UsageText()), unicode.IsSpace)
}

func (c *Command) ShortDescription() string {
	return c.cobra.Short
}

func (c *Command) Execute(args []string) error {
	defer profile.Measure("cobra:Execute", time.Now())
	c.logArgs(args)
	c.cobra.SetArgs(args)
	err := c.cobra.Execute()
	c.cobra.SetArgs(nil)
	rationalizeError(&err)
	return setupSensibleErrors(err, args)
}

func (c *Command) SetExamples(examples ...string) *Command {
	c.examples = append(c.examples, examples...)
	return c
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

func (c *Command) Unstable() bool {
	return c.unstable
}

func (c *Command) Examples() []string {
	return c.examples
}

func (c *Command) SetDescription(description string) {
	c.cobra.Short = description
}

func (c *Command) SetDisableFlagParsing(b bool) {
	c.cobra.DisableFlagParsing = b
}

func (c *Command) Name() string {
	return c.name
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

func (c *Command) ActiveFlagNames() []string {
	names := []string{}
	for _, flag := range c.ActiveFlags() {
		if flag.Name != "" {
			names = append(names, flag.Name)
		} else if flag.Shorthand != "" {
			names = append(names, flag.Shorthand)
		}
	}
	return names
}

func (c *Command) ActiveFlags() []*Flag {
	var flags []*Flag
	flagMapping := map[string]*Flag{}
	for _, flag := range c.flags {
		flagMapping[flag.Name] = flag
	}

	c.cobra.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed {
			return
		}
		if flag, ok := flagMapping[f.Name]; ok {
			flags = append(flags, flag)
		}
	})

	return flags
}

func (c *Command) ExecuteFunc() ExecuteFunc {
	return c.execute
}

func (c *Command) Arguments() []*Argument {
	return c.arguments
}

type ExecEventHandler func(cmd *Command, args []string) error

func (c *Command) OnExecStart(handler ExecEventHandler) {
	c.TopParent().onExecStart = append(c.TopParent().onExecStart, handler)
}

func (c *Command) OnExecStop(handler ExecEventHandler) {
	c.TopParent().onExecStop = append(c.TopParent().onExecStop, handler)
}

func DisableMousetrap() {
	cobra.MousetrapHelpText = ""
}

// SetUnstable denotes if the command as a beta feature. This will remove the command
// from state help, disable the commmand for those who haven't opted in to beta features,
// and add a warning banner for those who have.
func (c *Command) SetUnstable(unstable bool) *Command {
	c.unstable = unstable
	if !condition.OptInUnstable(c.cfg) {
		c.cobra.Hidden = unstable
	}
	return c
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

func (c *Command) DeprioritizeInHelpListing() {
	c.deprioritizeInHelpListing = true
}

// SetHasVariableArguments allows a captain Command to accept a variable number of command line
// arguments.
// By default, captain has Cobra restrict the command line arguments accepted to those given in the
// []*Argument list to NewCommand().
func (c *Command) SetHasVariableArguments() *Command {
	c.cobra.Args = nil
	return c
}

func (c *Command) SetSupportsStructuredOutput() *Command {
	c.structuredOutput = true
	return c
}

func (c *Command) SkipChecks() bool {
	return c.skipChecks
}

func (c *Command) SortBefore(c2 *Command) bool {
	switch {
	case c.group != c2.group:
		return c.group.SortBefore(c2.group)
	case c.deprioritizeInHelpListing == c2.deprioritizeInHelpListing:
		return c.Name() < c2.Name()
	default:
		return !c.deprioritizeInHelpListing
	}
}

func (c *Command) AddChildren(children ...*Command) {
	for _, child := range children {
		if c.unstable {
			child.SetUnstable(true)
		}

		c.commands = append(c.commands, child)
		c.cobra.AddCommand(child.cobra)

		if child.parent != nil {
			panic(fmt.Sprintf("Command %s already has a parent: %s", child.Name(), child.parent.Name()))
		}
		child.parent = c
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

func (c *Command) FindChild(args []string) (*Command, error) {
	foundCobra, _, err := c.cobra.Find(args)
	if err != nil {
		return nil, errs.Wrap(err, "Could not find child command with args: %s", strings.Join(args, " "))
	}
	if cmd, ok := cobraMapping[foundCobra]; ok {
		if cmd.parent == nil {
			// Cobra returns the parent command if no child was found, but we don't want that.
			return nil, nil
		}
		return cmd, nil
	}
	return nil, &ErrNoChildren{locale.NewError("err_captain_cmd_find", "Could not find child Command with args: {{.V0}}", strings.Join(args, " "))}
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
	if c.parent != nil {
		return c.parent.flagByName(name, persistOnly)
	}
	return nil
}

func (c *Command) persistRunner(cobraCmd *cobra.Command, args []string) {
	// Run OnUse functions for persistent flags
	c.runFlags(true)
}

// commandNames returns a slice of the names of the sub-commands called
func (c *Command) commandNames(includeRoot bool) []string {
	var commands []string
	cmd := c.cobra
	root := cmd.Root()
	for {
		if cmd == nil || (cmd == root && !includeRoot) {
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

// cobraExecHandler is the function that we've routed cobra to run when a command gets executed.
// It allows us to wrap some over-arching logic around command executions, and should never be called directly.
func (c *Command) cobraExecHandler(cobraCmd *cobra.Command, args []string) (rerr error) {
	defer profile.Measure("captain:runner", time.Now())

	subCommandString := c.JoinedSubCommandNames()
	rollbar.CurrentCmd = appEventPrefix + subCommandString

	// Send GA events unless they are handled in the runners...
	if c.analytics != nil {
		var label []string
		if len(args) > 0 && (args[0] == constants.PipShim) {
			label = append(label, args[0])
		}

		c.cobra.Flags().VisitAll(func(cobraFlag *pflag.Flag) {
			if !cobraFlag.Changed {
				return
			}

			var name string
			if cobraFlag.Name != "" {
				name = "--" + cobraFlag.Name
			} else {
				name = "-" + cobraFlag.Shorthand
			}
			label = append(label, name)
		})

		c.analytics.EventWithLabel(anaConsts.CatRunCmd, appEventPrefix+subCommandString, strings.Join(label, " "))

		if shim, got := os.LookupEnv(constants.ShimEnvVarName); got {
			c.analytics.Event(anaConsts.CatShim, shim)
		}
	}

	if c.shouldWarnUnstable() && !condition.OptInUnstable(c.cfg) {
		c.out.Notice(locale.Tr("unstable_command_warning"))
		return nil
	} else if c.out.Type().IsStructured() && !c.structuredOutput {
		return locale.NewInputError("err_no_structured_output", "", string(c.out.Type()))
	}

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

	c.outputTitleIfAny()

	// initialize signal handler for analytics events
	as := sighandler.NewAwaitingSigHandler(os.Interrupt)
	sighandler.Push(as)
	defer rtutils.Closer(sighandler.Pop, &rerr)

	err := as.WaitForFunc(func() error {
		defer profile.Measure("captain:cmd:execute", time.Now())

		for _, handler := range c.TopParent().onExecStart {
			if err := handler(c, args); err != nil {
				return errs.Wrap(err, "onExecStart handler failed")
			}
		}

		if err := c.execute(c, args); err != nil {
			if !locale.HasError(err) {
				return locale.WrapError(err, "unexpected_error", "Command failed due to unexpected error. For your convenience, this is the error chain:\n{{.V0}}", errs.JoinMessage(err))
			}
			return errs.Wrap(err, "execute failed")
		}

		for _, handler := range c.TopParent().onExecStop {
			if err := handler(c, args); err != nil {
				return errs.Wrap(err, "onExecStop handler failed")
			}
		}

		return nil
	})

	exitCode := errs.ParseExitCode(err)

	var serr interface{ Signal() os.Signal }
	if errors.As(err, &serr) {
		if c.analytics != nil {
			c.analytics.EventWithLabel(anaConsts.CatCommandExit, appEventPrefix+subCommandString, "interrupt")
		}
		err = locale.WrapInputError(err, "user_interrupt", "User interrupted the State Tool process.")
	} else {
		if c.analytics != nil {
			if err != nil && (subCommandString == "install" || subCommandString == "activate") {
				// This is a temporary hack; proper implementation: https://activestatef.atlassian.net/browse/DX-495
				c.analytics.EventWithLabel(anaConsts.CatCommandError, appEventPrefix+subCommandString, errs.JoinMessage(err))
			}
			c.analytics.EventWithLabel(anaConsts.CatCommandExit, appEventPrefix+subCommandString, strconv.Itoa(exitCode))
		}
	}

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

func (c *Command) shouldWarnUnstable() bool {
	return c.unstable && !c.out.Type().IsStructured()
}

func (c *Command) outputTitleIfAny() {
	if c.out != nil && c.title != "" {
		suffix := ""
		if c.unstable {
			suffix = locale.T("beta_suffix")
		}
		c.out.Notice(output.Title(c.title + suffix))
	}

	if c.shouldWarnUnstable() {
		c.out.Notice(locale.T("unstable_feature_banner"))
	}
}

// setupSensibleErrors inspects an error value for certain errors and returns a
// wrapped error that can be checked and that is localized.
func setupSensibleErrors(err error, args []string) error {
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

	// pflag error of the form "flag needs an argument: <flag>, called at: "
	if strings.Contains(errMsg, "flag needs an argument: ") {
		flag := strings.SplitN(errMsg, ": ", 2)[1]
		return locale.NewInputError(
			locale.Tl("command_flag_needs_argument", "Flag needs an argument: [NOTICE]{{.V0}}[/RESET]", flag))
	}

	if pflagErrFlag := pflagFlagErrMsgFlag(errMsg); pflagErrFlag != "" {
		return locale.NewInputError(
			"command_flag_no_such_flag",
			"No such flag: [NOTICE]{{.V0}}[/RESET]", pflagErrFlag,
		)
	}

	if pflagErrCmd := pflagCmdErrMsgCmd(errMsg); pflagErrCmd != "" {
		return locale.NewInputError("command_cmd_no_such_cmd", "", pflagErrCmd)
	}

	// Cobra error message of the form "accepts at most 0 arg(s), received 1, called at: "
	if strings.Contains(errMsg, "accepts at most ") {
		var max, received int
		n, err := fmt.Sscanf(errMsg, "accepts at most %d arg(s), received %d", &max, &received)
		if err != nil || n != 2 {
			multilog.Error("Unable to parse cobra error message: %v", err)
			return locale.NewInputError("err_cmd_unexpected_arguments", "Unexpected argument(s) given")
		}
		if max == 0 && received > 0 {
			return locale.NewInputError("command_cmd_no_such_cmd", "", args[len(args)-received])
		}
		return locale.NewInputError(
			"err_cmd_too_many_arguments",
			"Too many arguments given: {{.V0}} expected, {{.V1}} received",
			strconv.Itoa(max), strconv.Itoa(received))
	}

	return err
}

// pflag: flag.go: errors are not detectable
func pflagFlagErrMsgFlag(errMsg string) string {
	flagText := strings.TrimPrefix(errMsg, "no such flag ")
	flagText = strings.TrimPrefix(flagText, "unknown flag: ")
	flagText = strings.TrimPrefix(flagText, "bad flag syntax: ")
	// unknown shorthand flag: 'x' in -x
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

	contents, err := assets.ReadFileBytes("usage.tpl")
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}
	tpl, err = tpl.Parse(string(contents))
	if err != nil {
		return errs.Wrap(err, "Could not parse template")
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, map[string]interface{}{
		"Cmd":           cmd,
		"Cobra":         cmd.cobra,
		"OptinUnstable": condition.OptInUnstable(cmd.cfg),
	}); err != nil {
		return errs.Wrap(err, "Could not execute template")
	}

	if writer := cmd.cobra.OutOrStdout(); writer != os.Stdout {
		_, err := writer.Write(out.Bytes())
		if err != nil {
			return errs.Wrap(err, "Unable to write to cobra outWriter")
		}
	} else {
		cmd.out.Print(strings.TrimRightFunc(out.String(), unicode.IsSpace))
	}

	return nil

}

func childCommands(cmd *Command) string {
	if len(cmd.AvailableChildren()) == 0 {
		return ""
	}

	var group string
	table := table.New([]string{"", ""})
	table.HideHeaders = true
	for _, child := range cmd.AvailableChildren() {
		if group != child.Group().String() && child.Group().String() != "" {
			group = child.Group().String()
			table.AddRow([]string{""})
			table.AddRow([]string{fmt.Sprintf("%s:", group)})
		}
		if !child.cobra.Hidden {
			if child.unstable {
				table.AddRow([]string{fmt.Sprintf("  %s (Unstable)", child.Name()), child.ShortDescription()})
			} else {
				table.AddRow([]string{fmt.Sprintf("  %s", child.Name()), child.ShortDescription()})
			}
		}
	}

	return fmt.Sprintf("\n\nAvailable Commands:\n%s", table.Render())
}

func (c *Command) logArgs(args []string) {
	child, err := c.FindChild(args)
	if err != nil {
		logging.Debug("Could not find child command, error: %v", err)
	}

	var logArgs []string
	if child != nil {
		logArgs = append(logArgs, child.commandNames(false)...)
	}

	logging.Debug("Args: %s, Flags: %s", logArgs, flags(args))
}

func flags(args []string) []string {
	flags := []string{}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") || condition.InActiveStateCI() || condition.BuiltOnDevMachine() {
			flags = append(flags, arg)
		}
	}
	return flags
}
