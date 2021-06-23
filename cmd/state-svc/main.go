package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/runbits/panics"
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
		if panics.HandlePanics() {
			exitCode = 1
		}
		events.WaitForEvents(1*time.Second, rollbar.Close)
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

func run() error {
	var cmd command = ""
	if len(os.Args) > 1 {
		cmd = command(os.Args[1])
	}

	cfg, err := config.Get()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config")
	}

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

	// Handle sigterm
	sig := make(chan os.Signal, 1)
	shutdown := make(chan struct{})

	p := NewService(cfg, shutdown)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(sig)
		select {
		case oscall, ok := <-sig:
			if !ok {
				return
			}
			logging.Debug("system call:%+v", oscall)
		case <-shutdown:
		}
		if err := p.Stop(); err != nil {
			logging.Error("Stopping server failed: %v", errs.Join(err, ": "))
		}
	}()

	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	// Note: p.Start() returns before the server is completely shut down. Hence we need to wait for the shutdown process to complete:
	defer func() {
		signal.Stop(sig)
		close(shutdown)
		wg.Wait()
	}()

	if err := p.Start(); err != nil {
		return errs.Wrap(err, "Could not start service")
	}

	return nil
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
	if err := cfg.Reload(); err != nil {
		return errs.Wrap(err, "Could not reload configuration.")
	}
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
