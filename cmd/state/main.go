package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ActiveState/cli/cmd/state/internal/cmdtree"
	anAsync "github.com/ActiveState/cli/internal/analytics/client/async"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/prompt"
	_ "github.com/ActiveState/cli/internal/prompt" // Sets up survey defaults
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/svcctl"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	startTime := time.Now()

	var exitCode int
	// Set up logging
	rollbar.SetupRollbar(constants.StateToolRollbarToken)

	var cfg *config.Instance
	defer func() {
		// Handle panics gracefully, and ensure that we exit with non-zero code
		if panics.HandlePanics(recover(), debug.Stack()) {
			exitCode = 1
		}

		// ensure rollbar messages are called
		if err := events.WaitForEvents(5*time.Second, rollbar.Wait, authentication.LegacyClose, logging.Close); err != nil {
			logging.Warning("Failed waiting for events: %v", err)
		}

		if cfg != nil {
			events.Close("config", cfg.Close)
		}

		profile.Measure("main", startTime)

		// exit with exitCode
		os.Exit(exitCode)
	}()

	var err error
	cfg, err = config.New()
	if err != nil {
		multilog.Critical("Could not initialize config: %v", errs.JoinMessage(err))
		fmt.Fprintf(os.Stderr, "Could not load config, if this problem persists please reinstall the State Tool. Error: %s\n", errs.JoinMessage(err))
		exitCode = 1
		return
	}
	rollbar.SetConfig(cfg)

	// Set up our output formatter/writer
	outFlags := parseOutputFlags(os.Args)
	out, err := initOutput(outFlags, "")
	if err != nil {
		multilog.Critical("Could not initialize outputer: %s", errs.JoinMessage(err))
		os.Stderr.WriteString(locale.Tr("err_main_outputer", err.Error()))
		exitCode = 1
		return
	}

	// Set up our legacy outputer
	setPrinterColors(outFlags)

	isInteractive := strings.ToLower(os.Getenv(constants.NonInteractiveEnvVarName)) != "true" &&
		!outFlags.NonInteractive &&
		terminal.IsTerminal(int(os.Stdin.Fd())) &&
		out.Type() != output.EditorV0FormatName &&
		out.Type() != output.EditorFormatName
	// Run our main command logic, which is logic that defers to the error handling logic below
	err = run(os.Args, isInteractive, cfg, out)
	if err != nil {
		exitCode, err = unwrapError(err)
		if err != nil {
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
}

func run(args []string, isInteractive bool, cfg *config.Instance, out output.Outputer) (rerr error) {
	return errs.Wrap(
		locale.WrapError(
			errs.Wrap(
				locale.WrapInputError(
					errs.New("error 1"), "", "input error"),
				"error 2"),
			"", "local error"),
		"error 3")
	defer profile.Measure("main:run", time.Now())

	// Set up profiling
	if os.Getenv(constants.CPUProfileEnvVarName) != "" {
		cleanup, err := profile.CPU()
		if err != nil {
			return err
		}
		defer cleanup()
	}

	logging.CurrentHandler().SetVerbose(os.Getenv("VERBOSE") != "" || argsHaveVerbose(args))

	logging.Debug("ConfigPath: %s", cfg.ConfigPath())
	logging.Debug("CachePath: %s", storage.CachePath())

	svcExec, err := installation.ServiceExec()
	if err != nil {
		return errs.Wrap(err, "Could not get service info")
	}

	ipcClient := svcctl.NewDefaultIPCClient()
	svcPort, err := svcctl.EnsureExecStartedAndLocateHTTP(ipcClient, svcExec)
	if err != nil {
		return locale.WrapError(err, "start_svc_failed", "Failed to start state-svc at state tool invocation")
	}

	svcmodel := model.NewSvcModel(svcPort)

	// Retrieve project file
	pjPath, err := projectfile.GetProjectFilePath()
	if err != nil && errs.Matches(err, &projectfile.ErrorNoProjectFromEnv{}) {
		// Fail if we are meant to inherit the projectfile from the environment, but the file doesn't exist
		return err
	}

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

	pjNamespace := ""
	if pj != nil {
		pjNamespace = pj.Namespace().String()
	}

	auth := authentication.New(cfg)
	defer events.Close("auth", auth.Close)

	if err := auth.Sync(); err != nil {
		logging.Warning("Could not sync authenticated state: %s", err.Error())
	}

	an := anAsync.New(svcmodel, cfg, auth, out, pjNamespace)
	defer func() {
		if err := events.WaitForEvents(time.Second, an.Wait); err != nil {
			logging.Warning("Failed waiting for events: %v", err)
		}
	}()

	// Set up prompter
	prompter := prompt.New(isInteractive, an)

	// Set up conditional, which accesses a lot of primer data
	sshell := subshell.New(cfg)

	conditional := constraints.NewPrimeConditional(auth, pj, sshell.Shell())
	project.RegisterConditional(conditional)
	project.RegisterExpander("mixin", project.NewMixin(auth).Expander)
	project.RegisterExpander("secrets", project.NewSecretPromptingExpander(secretsapi.Get(), prompter, cfg))

	// Run the actual command
	cmds := cmdtree.New(primer.New(pj, out, auth, prompter, sshell, conditional, cfg, ipcClient, svcmodel, an), args...)

	childCmd, err := cmds.Command().Find(args[1:])
	if err != nil {
		logging.Debug("Could not find child command, error: %v", err)
	}

	if childCmd != nil && !childCmd.SkipChecks() {
		// Auto update to latest state tool version
		if updated, err := autoUpdate(args, cfg, out); err != nil || updated {
			return err
		}

		// Check for deprecation
		deprecationInfo, err := svcmodel.CheckDeprecation(context.Background())
		if err != nil {
			multilog.Error("Could not check for deprecation: %s", err.Error())
		}
		if deprecationInfo != nil {
			if !deprecationInfo.DateReached {
				out.Notice(output.Heading(locale.Tl("deprecation_title", "Deprecation Warning")))
				out.Notice(locale.Tr("warn_deprecation", deprecationInfo.Date, deprecationInfo.Reason))
			} else {
				return locale.NewInputError("err_deprecation", "You are running a version of the State Tool that is no longer supported! Reason: {{.V1}}", deprecationInfo.Date, deprecationInfo.Reason)
			}
		}

		if childCmd.Name() != "update" && pj != nil && pj.IsLocked() {
			if (pj.Version() != "" && pj.Version() != constants.Version) ||
				(pj.VersionBranch() != "" && pj.VersionBranch() != constants.BranchName) {
				return errs.AddTips(
					locale.NewInputError("lock_version_mismatch", "", pj.Source().Lock, constants.BranchName, constants.Version),
					locale.Tl("lock_update_legacy_version", "", constants.DocumentationURLLocking),
					locale.T("lock_update_lock"),
				)
			}
		}
	}

	err = cmds.Execute(args[1:])
	if err != nil {
		cmdName := ""
		if childCmd != nil {
			cmdName = childCmd.UseFull() + " "
		}
		err = errs.AddTips(err, locale.Tl("err_tip_run_help", "Run → [ACTIONABLE]`state {{.V0}}--help`[/RESET] for general help", cmdName))
	}

	return err
}

func argsHaveVerbose(args []string) bool {
	var isRunOrExec bool
	nextArg := 0

	for i, arg := range args {
		if arg == "run" || arg == "exec" {
			isRunOrExec = true
			nextArg = i + 1
		}

		// Skip looking for verbose args after --, eg. for `state shim -- perl -v`
		if arg == "--" {
			return false
		}
		if (arg == "--verbose" || arg == "-v") && (!isRunOrExec || i == nextArg) {
			return true
		}
	}
	return false
}
