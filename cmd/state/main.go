package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ActiveState/cli/cmd/state/internal/cmdtree"
	"github.com/ActiveState/cli/cmd/state/internal/cmdtree/exechandlers/messenger"
	anAsync "github.com/ActiveState/cli/internal/analytics/client/async"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/migrator"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/prompt"
	_ "github.com/ActiveState/cli/internal/prompt" // Sets up survey defaults
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits/errors"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/svcctl"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func main() {
	startTime := time.Now()

	var exitCode int
	// Set up logging
	rollbar.SetupRollbar(constants.StateToolRollbarToken)

	// We have to disable mouse trap as without it the state:// protocol cannot work
	captain.DisableMousetrap()

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

	// Configuration options
	// This should only be used if the config option is not exclusive to one package.
	configMediator.RegisterOption(constants.OptinBuildscriptsConfig, configMediator.Bool, false)

	// Set up our output formatter/writer
	outFlags := parseOutputFlags(os.Args)
	shellName, _ := subshell.DetectShell(cfg)
	out, err := initOutput(outFlags, "", shellName)
	if err != nil {
		multilog.Critical("Could not initialize outputer: %s", errs.JoinMessage(err))
		os.Stderr.WriteString(locale.Tr("err_main_outputer", err.Error()))
		exitCode = 1
		return
	}

	// Set up our legacy outputer
	setPrinterColors(outFlags)

	isInteractive := strings.ToLower(os.Getenv(constants.NonInteractiveEnvVarName)) != "true" && out.Config().Interactive
	// Run our main command logic, which is logic that defers to the error handling logic below
	err = run(os.Args, isInteractive, cfg, out)
	if err != nil {
		exitCode, err = errors.ParseUserFacing(err)
		if err != nil {
			out.Error(err)
		}
	}
}

func run(args []string, isInteractive bool, cfg *config.Instance, out output.Outputer) (rerr error) {
	defer profile.Measure("main:run", time.Now())

	// Set up profiling
	if os.Getenv(constants.CPUProfileEnvVarName) != "" {
		cleanup, err := profile.CPU()
		if err != nil {
			return err
		}
		defer rtutils.Closer(cleanup, &rerr)
	}

	logging.CurrentHandler().SetVerbose(os.Getenv("VERBOSE") != "" || argsHaveVerbose(args))

	logging.Debug("ConfigPath: %s", cfg.ConfigPath())
	logging.Debug("CachePath: %s", storage.CachePath())

	svcExec, err := installation.ServiceExec()
	if err != nil {
		return errs.Wrap(err, "Could not get service info")
	}

	ipcClient := svcctl.NewDefaultIPCClient()
	argText := strings.Join(args, " ")
	svcPort, err := svcctl.EnsureExecStartedAndLocateHTTP(ipcClient, svcExec, argText, out)
	if err != nil {
		return locale.WrapError(err, "start_svc_failed", "Failed to start state-svc at state tool invocation")
	}

	svcmodel := model.NewSvcModel(svcPort)

	// Amend Rollbar data to also send the state-svc log tail. This cannot be done inside the rollbar
	// package itself because importing pkg/platform/model creates an import cycle.
	rollbar.AddLogDataAmender(func(logData string) string {
		ctx, cancel := context.WithTimeout(context.Background(), model.SvcTimeoutMinimal)
		defer cancel()
		svcLogData, err := svcmodel.FetchLogTail(ctx)
		if err != nil {
			svcLogData = fmt.Sprintf("Could not fetch state-svc log: %v", err)
		}
		logData += "\nstate-svc log:\n"
		if len(svcLogData) == logging.TailSize {
			logData += "<truncated>\n"
		}
		logData += svcLogData
		return logData
	})

	auth := authentication.New(cfg)
	defer events.Close("auth", auth.Close)

	if err := auth.Sync(); err != nil {
		logging.Warning("Could not sync authenticated state: %s", errs.JoinMessage(err))
	}

	projectfile.RegisterMigrator(migrator.NewMigrator(auth, cfg))

	// Retrieve project file
	if os.Getenv("ACTIVESTATE_PROJECT") != "" {
		out.Notice(locale.T("warning_activestate_project_env_var"))
	}
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

	an := anAsync.New(anaConst.SrcStateTool, svcmodel, cfg, auth, out, pjNamespace)
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
	if err := project.RegisterExpander("mixin", project.NewMixin(auth).Expander); err != nil {
		logging.Debug("Could not register mixin expander: %v", err)
	}

	if err := project.RegisterExpander("secrets", project.NewSecretPromptingExpander(secretsapi.Get(auth), prompter, cfg, auth)); err != nil {
		logging.Debug("Could not register secrets expander: %v", err)
	}

	// Run the actual command
	cmds := cmdtree.New(primer.New(pj, out, auth, prompter, sshell, conditional, cfg, ipcClient, svcmodel, an), args...)

	childCmd, err := cmds.Command().FindChild(args[1:])
	if err != nil {
		logging.Debug("Could not find child command, error: %v", err)
	}

	msger := messenger.New(out, svcmodel)
	cmds.OnExecStart(msger.OnExecStart)
	cmds.OnExecStop(msger.OnExecStop)

	if childCmd != nil && !childCmd.SkipChecks() && !out.Type().IsStructured() {
		// Auto update to latest state tool version
		if updated, err := autoUpdate(svcmodel, args, cfg, an, out); err == nil && updated {
			return nil // command will be run by updated exe
		} else if err != nil {
			multilog.Error("Failed to autoupdate: %v", err)
		}

		if childCmd.Name() != "update" && pj != nil && pj.IsLocked() {
			if (pj.Version() != "" && pj.Version() != constants.Version) ||
				(pj.Channel() != "" && pj.Channel() != constants.ChannelName) {
				return errs.AddTips(
					locale.NewInputError("lock_version_mismatch", "", pj.Source().Lock, constants.ChannelName, constants.Version),
					locale.Tr("lock_update_legacy_version", constants.DocumentationURLLocking),
					locale.T("lock_update_lock"),
				)
			}
		}
	}

	err = cmds.Execute(args[1:])
	if err != nil {
		cmdName := ""
		if childCmd != nil {
			cmdName = childCmd.JoinedSubCommandNames() + " "
		}
		if !out.Type().IsStructured() {
			err = errs.AddTips(err, locale.Tl("err_tip_run_help", "Run → '[ACTIONABLE]state {{.V0}}--help[/RESET]' for general help", cmdName))
		}
		errors.ReportError(err, cmds.Command(), an)
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
