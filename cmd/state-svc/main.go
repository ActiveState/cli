package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ActiveState/cli/cmd/state-svc/internal/server"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, errs.Join(err, ": ").Error())
		os.Exit(1)
	}
}

func run() error {
	s, err := server.NewServer()
	if err != nil {
		return errs.Wrap(err, "Could not create server")
	}

	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config")
	}
	if err := cfg.Set("port", s.Port()); err != nil {
		return errs.Wrap(err, "Could not save config")
	}

	// Handle sigterm
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		oscall := <-c
		logging.Debug("system call:%+v", oscall)
		if err := s.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Closing server failed: %w", err)
		}
	}()

	if err := s.Start(); err != nil {
		return errs.Wrap(err, "Failed to start server")
	}

	return nil
}
