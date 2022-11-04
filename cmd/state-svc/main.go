package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime/debug"
	"syscall"
	"time"

	anaSync "github.com/ActiveState/cli/internal/analytics/client/sync"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/inconshreveable/mousetrap"
)

const (
	cmdStart      = "start"
	cmdStop       = "stop"
	cmdStatus     = "status"
	cmdForeground = "foreground"
)

func main() {
	var exitCode int

	var cfg *config.Instance
	defer func() {
		if panics.HandlePanics(recover(), debug.Stack()) {
			exitCode = 1
		}

		if cfg != nil {
			events.Close("config", cfg.Close)
		}

		if err := events.WaitForEvents(5*time.Second, rollbar.Wait, authentication.LegacyClose, logging.Close); err != nil {
			logging.Warning("Failing to wait events")
		}
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
	rollbar.SetupRollbar(constants.StateServiceRollbarToken)
	rollbar.SetConfig(cfg)

	if os.Getenv("VERBOSE") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}

	runErr := run(cfg)
	if runErr != nil {
		errMsg := errs.Join(runErr, ": ").Error()
		if locale.IsInputError(runErr) {
			logging.Debug("state-svc errored out due to input: %s", errMsg)
		} else {
			multilog.Critical("state-svc errored out: %s", errMsg)
		}

		fmt.Fprintln(os.Stderr, errMsg)
		exitCode = 1
	}
}

func run(cfg *config.Instance) error {
	args := os.Args

	auth := authentication.New(cfg)
	an := anaSync.New(cfg, auth)
	defer an.Wait()

	out, err := output.New("", &output.Config{
		OutWriter: os.Stdout,
		ErrWriter: os.Stderr,
	})
	if err != nil {
		return errs.Wrap(err, "Could not initialize outputer")
	}

	if mousetrap.StartedByExplorer() {
		// Allow starting the svc via a double click
		captain.DisableMousetrap()
		return runStart(out, "svc-start:mouse")
	}

	p := primer.New(nil, out, nil, nil, nil, nil, cfg, nil, nil, an)

	cmd := captain.NewCommand(
		path.Base(os.Args[0]), "", "", p, nil, nil,
		func(ccmd *captain.Command, args []string) error {
			out.Print(ccmd.UsageText())
			return nil
		},
	)

	var foregroundArgText string

	cmd.AddChildren(
		captain.NewCommand(
			cmdStart,
			"Starting the ActiveState Service",
			"Start the ActiveState Service (Background)",
			p, nil, nil,
			func(ccmd *captain.Command, args []string) error {
				logging.Debug("Running CmdStart")
				return runStart(out, "svc-start:cli")
			},
		),
		captain.NewCommand(
			cmdStop,
			"Stopping the ActiveState Service",
			"Stop the ActiveState Service",
			p, nil, nil,
			func(ccmd *captain.Command, args []string) error {
				logging.Debug("Running CmdStop")
				return runStop()
			},
		),
		captain.NewCommand(
			cmdStatus,
			"Checking the ActiveState Service",
			"Display the Status of the ActiveState Service",
			p, nil, nil,
			func(ccmd *captain.Command, args []string) error {
				logging.Debug("Running CmdStatus")
				return runStatus(out)
			},
		),
		captain.NewCommand(
			cmdForeground,
			"Starting the ActiveState Service",
			"Start the ActiveState Service (Foreground)",
			p, nil,
			[]*captain.Argument{
				{
					Name:        "Arg text",
					Description: "Argument text of calling process to be reported if this application is started too often",
					Value:       &foregroundArgText,
				},
			},
			func(ccmd *captain.Command, args []string) error {
				logging.Debug("Running CmdForeground")
				if err := auth.Sync(); err != nil {
					logging.Warning("Could not sync authenticated state: %s", err.Error())
				}
				return runForeground(cfg, an, auth, foregroundArgText)
			},
		),
	)

	return cmd.Execute(args[1:])
}

func runForeground(cfg *config.Instance, an *anaSync.Client, auth *authentication.Auth, argText string) error {
	logging.Debug("Running in Foreground")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logFile := logging.FilePath()
	logging.Debug("Logging to %q", logFile)

	p := NewService(ctx, cfg, an, auth, logFile)

	if argText != "" {
		argText = fmt.Sprintf(" (invoked by %q)", argText)
	}

	if err := p.Start(); err != nil {
		return errs.Wrap(err, "Could not start service"+argText)
	}

	// Handle sigterm
	sig := make(chan os.Signal, 1)
	go func() {
		defer close(sig)

		select {
		case oscall, ok := <-sig:
			if !ok {
				return
			}
			logging.Debug("system call:%+v", oscall)
			// issue a service shutdown on interrupt
			cancel()
			if err := p.Stop(); err != nil {
				logging.Debug("Service stop failed: %v", err)
			}
		case <-ctx.Done():
		}
	}()
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	p.RunIfNotAuthority(time.Second*3, svcctl.NewDefaultIPCClient(), func(err error) {
		fmt.Fprintln(os.Stderr, err)

		cancel()
		if err := p.Stop(); err != nil {
			multilog.Critical("Service stop failed: %v", errs.JoinMessage(err))
		}
	})

	if err := p.Wait(); err != nil {
		return errs.Wrap(err, "Failure while waiting for server stop")
	}

	return nil
}

func runStart(out output.Outputer, argText string) error {
	if _, err := svcctl.EnsureStartedAndLocateHTTP(argText); err != nil {
		if errors.Is(err, ipc.ErrInUse) {
			out.Print("A State Service instance is already running in the background.")
			return nil
		}
		return errs.Wrap(err, "Could not start serviceManager")
	}

	return nil
}

func runStop() error {
	ipcClient := svcctl.NewDefaultIPCClient()
	if err := svcctl.StopServer(ipcClient); err != nil {
		return errs.Wrap(err, "Could not stop serviceManager")
	}
	return nil
}

func runStatus(out output.Outputer) error {
	ipcClient := svcctl.NewDefaultIPCClient()
	// Don't run in background if we're already running
	port, err := svcctl.LocateHTTP(ipcClient)
	if err != nil {
		return errs.Wrap(err, "Service cannot be reached")
	}
	out.Print(fmt.Sprintf("Port: %s", port))
	out.Print(fmt.Sprintf("Dashboard: http://127.0.0.1%s", port))

	logfile, err := svcctl.LogFileName(ipcClient)
	if err != nil {
		return errs.Wrap(err, "Service could not locate log file")
	}
	out.Print(fmt.Sprintf("Log: %s", logging.FilePathFor(logfile)))

	return nil
}
