package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/logging"
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
	logging.SetupRollbar(constants.StateServiceRollbarToken)
	defer exit(exitCode)

	if os.Getenv("VERBOSE") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}

	err := run()
	if err != nil {
		errMsg := errs.Join(err, ": ").Error()
		logging.Errorf("state-svc errored out: %s", errMsg)
		fmt.Fprintln(os.Stderr, errMsg)
		exitCode = 1
	}
}

func run() error {
	var cmd command = ""
	if len(os.Args) > 1 {
		cmd = command(os.Args[1])
	}

	cfg, err := config.New()
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

	p := NewService(cfg)

	// Handle sigterm
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	go func() {
		oscall := <-sig
		logging.Debug("system call:%+v", oscall)
		if err := p.Stop(); err != nil {
			logging.Error("Stop on sigterm failed: %v", errs.Join(err, ": "))
		}
	}()

	if err := p.Start(); err != nil {
		return errs.Wrap(err, "Could not start service")
	}

	return nil
}

func runStart(cfg *config.Instance) error {
	s := NewServiceManager(cfg)
	if err := s.Start(os.Args[0], CmdForeground); err != nil {
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
	pid, err := NewServiceManager(cfg).Pid()
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

func exit(code int) {
	events.WaitForEvents(1*time.Second, rollbar.Close)
	os.Exit(code)
}
