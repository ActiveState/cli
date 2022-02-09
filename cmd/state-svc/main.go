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

	anaSvc "github.com/ActiveState/cli/internal/analytics/client/sync"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/rollbar/rollbar-go"
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

		if err := cfg.Close(); err != nil {
			logging.Error("Failed to close config: %w", err)
		}

		if err := events.WaitForEvents(5*time.Second, rollbar.Wait, rollbar.Close, authentication.LegacyClose, logging.Close); err != nil {
			logging.Warning("Failing to wait events")
		}
		os.Exit(exitCode)
	}()

	cfg, err := config.New()
	if err != nil {
		logging.Critical("Could not initialize config: %v", errs.JoinMessage(err))
		fmt.Fprintf(os.Stderr, "Could not load config, if this problem persists please reinstall the State Tool. Error: %s\n", errs.JoinMessage(err))
		exitCode = 1
		return
	}
	logging.CurrentHandler().SetConfig(cfg)
	logging.SetupRollbar(constants.StateServiceRollbarToken)

	if os.Getenv("VERBOSE") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}

	runErr := run(cfg)
	if runErr != nil {
		errMsg := errs.Join(runErr, ": ").Error()
		if locale.IsInputError(runErr) {
			logging.Debug("state-svc errored out due to input: %s", errMsg)
		} else {
			logging.Critical("state-svc errored out: %s", errMsg)
		}

		fmt.Fprintln(os.Stderr, errMsg)
		exitCode = 1
	}
}

func run(cfg *config.Instance) (rerr error) {
	args := os.Args

	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config")
	}
	defer rtutils.Closer(cfg.Close, &rerr)

	machineid.Configure(cfg)
	machineid.SetErrorLogger(logging.Error)
	an := anaSvc.New(cfg, authentication.LegacyGet())
	defer an.Wait()

	out, err := output.New("", &output.Config{
		OutWriter: os.Stdout,
		ErrWriter: os.Stderr,
	})

	p := primer.New(nil, out, nil, nil, nil, nil, cfg, nil, nil, an)

	cmd := captain.NewCommand(
		path.Base(os.Args[0]), "", "", p, nil, nil,
		func(ccmd *captain.Command, args []string) error {
			fmt.Println("top level")
			return nil
		},
	)

	cmd.AddChildren(
		captain.NewCommand(
			cmdStart,
			"Starting the ActiveState Service",
			"Start the ActiveState Service (Background)",
			p, nil, nil,
			func(ccmd *captain.Command, args []string) error {
				logging.Debug("Running CmdStart")
				return runStart(cfg)
			},
		),
		captain.NewCommand(
			cmdStop,
			"Stopping the ActiveState Service",
			"Stop the ActiveState Service",
			p, nil, nil,
			func(ccmd *captain.Command, args []string) error {
				logging.Debug("Running CmdStop")
				return runStop(cfg)
			},
		),
		captain.NewCommand(
			cmdStatus,
			"Starting the ActiveState Service",
			"Display the Status of the ActiveState Service",
			p, nil, nil,
			func(ccmd *captain.Command, args []string) error {
				logging.Debug("Running CmdStatus")
				return runStatus(cfg)
			},
		),
		captain.NewCommand(
			cmdForeground,
			"Starting the ActiveState Service",
			"Start the ActiveState Service (Foreground)",
			p, nil, nil,
			func(ccmd *captain.Command, args []string) error {
				logging.Debug("Running CmdForeground")
				return runForeground(cfg, an)
			},
		),
	)

	return cmd.Execute(args[1:])
}

func runForeground(cfg *config.Instance, an *anaSvc.Client) error {
	logging.Debug("Running in Foreground")

	// create a global context for the service: When cancelled we issue a shutdown here, and wait for it to finish
	ctx, shutdown := context.WithCancel(context.Background())
	p := NewService(cfg, an, shutdown)

	// Handle sigterm
	sig := make(chan os.Signal, 1)
	go func() {
		defer close(sig)
		oscall, ok := <-sig
		if !ok {
			return
		}
		logging.Debug("system call:%+v", oscall)
		// issue a service shutdown on interrupt
		shutdown()
	}()
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	serverErr := make(chan error)
	go func() {
		err := p.Start()
		if err != nil {
			err = errs.Wrap(err, "Could not start service")
		}

		serverErr <- err
	}()

	// cancellation of context issues server shutdown
	<-ctx.Done()
	if err := p.Stop(); err != nil {
		return errs.Wrap(err, "Failed to stop service")
	}

	err := <-serverErr
	return err
}

func runStart(cfg *config.Instance) error {
	s := NewServiceManager(cfg)
	if err := s.Start(os.Args[0], cmdForeground); err != nil {
		if errors.Is(err, ErrSvcAlreadyRunning) {
			fmt.Println("A State Service instance is already running in the background.")
			return nil
		}
		return errs.Wrap(err, "Could not start serviceManager")
	}

	return nil
}

func runStop(cfg *config.Instance) error {
	s := NewServiceManager(cfg)
	if err := s.Stop(); err != nil {
		return errs.Wrap(err, "Could not stop serviceManager")
	}

	return nil
}

func runStatus(cfg *config.Instance) error {
	pid, err := NewServiceManager(cfg).CheckPid(cfg.GetInt(constants.SvcConfigPid))
	if err != nil {
		return errs.Wrap(err, "Could not obtain pid")
	}

	if pid == nil {
		fmt.Println("Service is not running")
		return nil
	}

	// Don't run in background if we're already running
	port := cfg.GetInt(constants.SvcConfigPort)

	fmt.Printf("Pid: %d\n", *pid)
	fmt.Printf("Port: %d\n", port)
	fmt.Printf("Dashboard: http://127.0.0.1:%d\n", port)
	fmt.Printf("Log: %s\n", logging.FilePathFor(logging.FileNameFor(*pid)))

	return nil
}
