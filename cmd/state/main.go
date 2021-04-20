package main

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/ActiveState/sysinfo"
	"github.com/rollbar/rollbar-go"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/ActiveState/cli/cmd/state/internal/cmdtree"
	"github.com/ActiveState/cli/internal/config" // MUST be first!
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/deprecation"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/prompt"
	_ "github.com/ActiveState/cli/internal/prompt" // Sets up survey defaults
	"github.com/ActiveState/cli/internal/subshell"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func main() {
	// Set up logging
	logging.SetupRollbar(constants.StateToolRollbarToken)
	defer rollbar.Close()

	// Handle panics gracefully
	defer handlePanics(os.Exit)

	// Set up our output formatter/writer
	outFlags := parseOutputFlags(os.Args)
	out, err := initOutput(outFlags, "")
	if err != nil {
		os.Stderr.WriteString(locale.Tr("err_main_outputer", err.Error()))
		os.Exit(1)
	}

	if runtime.GOOS == "windows" {
		osv, err := sysinfo.OSVersion()
		if err != nil {
			logging.Debug("Could not retrieve os version info: %v", err)
		} else if osv.Major < 10 {
			out.Notice(output.Heading(locale.Tl("compatibility_warning", "Compatibility Warning")))
			out.Notice(locale.Tr(
				"windows_compatibility_warning",
				constants.ForumsURL,
			))
		}
	}

	// Set up our legacy outputer
	setPrinterColors(outFlags)

	isInteractive := strings.ToLower(os.Getenv(constants.NonInteractive)) != "true" &&
		!outFlags.NonInteractive &&
		terminal.IsTerminal(int(os.Stdin.Fd()))
	// Run our main command logic, which is logic that defers to the error handling logic below
	code := 0
	err = run(os.Args, isInteractive, out)
	if err != nil {
		code, err = unwrapError(err)
		if !isSilent(err) {
			out.Error(err)
		}

		// If a state tool error occurs in a VSCode integrated terminal, we want
		// to pause and give time to the user to read the error message.
		// But not, if we exit, because the last command in the activated sub-shell failed.
		var eerr *exec.ExitError
		isExitError := errors.As(err, &eerr)
		if !isExitError && outFlags.ConfirmExit {
			out.Print(locale.T("confirm_exit_on_error_prompt"))
			br := bufio.NewReader(os.Stdin)
			br.ReadLine()
		}
	}

	os.Exit(code)
}

func run(args []string, isInteractive bool, out output.Outputer) error {
	// Set up profiling
	if os.Getenv(constants.CPUProfileEnvVarName) != "" {
		cleanup, err := profile.CPU()
		if err != nil {
			return err
		}
		defer cleanup()
	}

	verbose := os.Getenv("VERBOSE") != "" || argsHaveVerbose(args)
	logging.CurrentHandler().SetVerbose(verbose)

	cfg, err := config.Get()
	if err != nil {
		return locale.WrapError(err, "config_get_error", "Failed to load configuration.")
	}
	logging.Debug("ConfigPath: %s", cfg.ConfigPath())
	logging.Debug("CachePath: %s", cfg.CachePath())

	// set global configuration instances
	machineid.SetConfiguration(cfg)
	machineid.SetErrorLogger(logging.Error)
	logging.UpdateConfig(cfg)

	// Retrieve project file
	pjPath, err := projectfile.GetProjectFilePath()
	if err != nil && errs.Matches(err, &projectfile.ErrorNoProjectFromEnv{}) {
		// Fail if we are meant to inherit the projectfile from the environment, but the file doesn't exist
		return err
	}

	// Set up prompter
	prompter := prompt.New(isInteractive)

	// Set up project (if we have a valid path)
	var pj *project.Project
	if pjPath != "" {
		pjf, err := projectfile.FromPath(pjPath)
		if err != nil {
			return err
		}
		pj, err = project.New(pjf, out)
		if err != nil {
			return err
		}
	}

	// Forward call to specific state tool version, if warranted
	forward, err := forwardFn(cfg.ConfigPath(), args, out, pj)
	if err != nil {
		return err
	}
	if forward != nil {
		return forward()
	}

	pjOwner := ""
	pjNamespace := ""
	pjName := ""
	if pj != nil {
		pjOwner = pj.Owner()
		pjNamespace = pj.Namespace().String()
		pjName = pj.Name()
	}
	// Set up conditional, which accesses a lot of primer data
	sshell := subshell.New(cfg)
	auth := authentication.Get()
	conditional := constraints.NewPrimeConditional(auth, pjOwner, pjName, pjNamespace, sshell.Shell())
	project.RegisterConditional(conditional)
	project.RegisterExpander("mixin", project.NewMixin(auth).Expander)
	project.RegisterExpander("secrets", project.NewSecretPromptingExpander(secretsapi.Get(), prompter, cfg))

	// Run the actual command
	cmds := cmdtree.New(primer.New(pj, out, auth, prompter, sshell, conditional, cfg), args...)

	childCmd, err := cmds.Command().Find(args[1:])
	if err != nil {
		logging.Debug("Could not find child command, error: %v", err)
	}

	if childCmd != nil && !childCmd.SkipChecks() {
		// Check for deprecation
		deprecated, err := deprecation.Check(cfg)
		if err != nil {
			logging.Error("Could not check for deprecation: %s", err.Error())
		}
		if deprecated != nil {
			date := deprecated.Date.Format(constants.DateFormatUser)
			if !deprecated.DateReached {
				out.Notice(output.Heading(locale.Tl("deprecation_title", "Deprecation Warning")))
				out.Notice(locale.Tr("warn_deprecation", date, deprecated.Reason))
			} else {
				return locale.NewInputError("err_deprecation", "You are running a version of the State Tool that is no longer supported! Reason: {{.V1}}", date, deprecated.Reason)
			}
		}
	}

	err = cmds.Execute(args[1:])
	if err != nil {
		cmdName := ""
		if childCmd != nil {
			cmdName = childCmd.Use() + " "
		}
		err = errs.AddTips(err, locale.Tl("err_tip_run_help", "Run → [ACTIONABLE]`state {{.V0}}--help`[/RESET] for general help", cmdName))
	}

	if childCmd == nil || !childCmd.SkipChecks() {
		// Auto update to latest state tool version, only runs once per day
		// Todo: This is better done in the `state-svc` process  https://www.pivotaltracker.com/story/show/177730748
		if updated, err := autoUpdate(args, cfg, pjPath); err != nil || updated {
			return err
		}
	}

	return err
}

func argsHaveVerbose(args []string) bool {
	for _, arg := range args {
		// Skip looking for verbose args after --, eg. for `state shim -- perl -v`
		if arg == "--" {
			return false
		}
		if arg == "--verbose" || arg == "-v" {
			return true
		}
	}
	return false
}
