package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/analytics/client/sync"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/ptr"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/cmdlets/errors"
)

func main() {
	var exitCode int

	var an analytics.Dispatcher
	var cfg *config.Instance
	rollbar.SetupRollbar(constants.OfflineInstallerRollbarToken)

	// Allow starting the installer via a double click
	captain.DisableMousetrap()

	// Handle things like panics, exit codes and the closing of globals
	defer func() {
		if panics.HandlePanics(recover(), debug.Stack()) {
			exitCode = 1
		}

		if err := cfg.Close(); err != nil {
			logging.Error("Failed to close config: %w", err)
		}

		if err := events.WaitForEvents(5*time.Second, rollbar.Wait, an.Wait, logging.Close); err != nil {
			logging.Warning("state-remote-installer failed to wait for events: %v", err)
		}
		os.Exit(exitCode)
	}()

	if os.Getenv("VERBOSE") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}

	// Set up configuration handler
	cfg, err := config.New()
	if err != nil {
		logging.Critical("Could not set up configuration handler: " + errs.JoinMessage(err))
		fmt.Fprintln(os.Stderr, errs.JoinMessage(err))
		exitCode = 1
		return
	}

	rollbar.SetConfig(cfg)
	an = sync.New(cfg, nil)

	out, err := output.New("", &output.Config{
		OutWriter: os.Stdout,
		ErrWriter: os.Stderr,
	})
	if err != nil {
		logging.Critical("Could not set up outputter: " + errs.JoinMessage(err))
		fmt.Fprintln(os.Stderr, errs.JoinMessage(err))
		exitCode = 1
		return
	}

	prime := primer.New(
		nil, out, nil,
		prompt.New(true, an),
		subshell.New(cfg), nil, cfg,
		nil, nil, an)

	if err := run(prime); err != nil {
		if locale.IsInputError(err) {
			logging.Debug("state-offline-uninstaller errored out due to input: %s", errs.JoinMessage(err))
		} else {
			multilog.Critical("state-offline-uninstaller errored out: %s", errs.JoinMessage(err))
		}

		exitCode, _ = errors.Unwrap(err)
		if err != nil {
			fmt.Fprintln(os.Stderr, errs.JoinMessage(err))
		}
	}
	out.Print("Press enter to exit.")
	fmt.Scanln(ptr.StrP("")) // Wait for input from user
}

func run(prime *primer.Values) error {
	params := newParams()
	cmd := captain.NewCommand(
		"uninstall",
		"Doing offline un-installation",
		"Do an offline un-installation",
		prime, nil,
		[]*captain.Argument{
			{
				Name:        "path",
				Description: "Directory to uninstall <path>",
				Value:       &params.path,
				Required:    false,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			logging.Debug("Running CmdUnInstall")
			runner := NewRunner(prime)
			return runner.Run(params)
		},
	)

	err := cmd.Execute(os.Args[1:])
	if err != nil {
		errors.PanicOnMissingLocale = false
		errors.ReportError(err, cmd, prime.Analytics())
		return err
	}

	return nil
}
