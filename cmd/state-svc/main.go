package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/rollbar/rollbar-go"
)

type command string

const (
	CmdStart      = "start"
	CmdStop       = "stop"
	CmdStatus     = "status"
	CmdForeground = "foreground"
)

var commands = []command{
	CmdStart,
	CmdStop,
	CmdStatus,
	CmdForeground,
}

func main() {
	var exitCode int
	defer func() {
		if panics.HandlePanics(recover()) {
			exitCode = 1
		}
		if err := events.WaitForEvents(1*time.Second, rollbar.Close, authentication.LegacyClose); err != nil {
			logging.Warning("Failing to wait for rollbar to close")
		}
		os.Exit(exitCode)
	}()

	logging.SetupRollbar(constants.StateServiceRollbarToken)

	if os.Getenv("VERBOSE") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}

	err := run()
	if err != nil {
		errMsg := errs.Join(err, ": ").Error()
		logger := logging.Error
		if locale.IsInputError(err) {
			logger = logging.Debug
		}
		logger("state-svc errored out: %s", errMsg)

		fmt.Fprintln(os.Stderr, errMsg)
		exitCode = 1
	}
}

func run() (rerr error) {
	var cmd command = ""
	if len(os.Args) > 1 {
		cmd = command(os.Args[1])
	}

	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config")
	}
	defer rtutils.Closer(cfg.Close, &rerr)

	analytics.Configure(cfg)
	machineid.Configure(cfg)
	machineid.SetErrorLogger(logging.Error)

	switch cmd {
	case CmdStart:
		logging.Debug("Running CmdStart")
		return runStart(cfg)
	case CmdStop:
		logging.Debug("Running CmdStop")
		return runStop(cfg)
	case CmdStatus:
		logging.Debug("Running CmdStatus")
		return runStatus(cfg)
	case CmdForeground:
		logging.Debug("Running CmdForeground")
		return runForeground(cfg)
	}

	return errs.New("Expected one of following commands: %v", commands)
}

func runForeground(cfg *config.Instance) error {
	logging.Debug("Running in Foreground")

	// create a global context for the service: When cancelled we issue a shutdown here, and wait for it to finish
	ctx, shutdown := context.WithCancel(context.Background())
	p := NewService(cfg, shutdown)

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
	if err := s.Start(os.Args[0], CmdForeground); err != nil {
		if errors.Is(err, ErrSvcAlreadyRunning) {
			err = locale.WrapInputError(err, "svc_start_already_running_err", "A State Service instance is already running in the background.")
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
