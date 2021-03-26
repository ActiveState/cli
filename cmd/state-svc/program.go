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

type program struct {
	server *server.Server
	sig    chan os.Signal
}

func NewProgram() *program {
	return &program{}
}

func (p *program) Start() error {
	logging.Debug("program:Start")

	var err error
	p.server, err = server.New()
	if err != nil {
		return errs.Wrap(err, "Could not create server")
	}

	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config")
	}
	if err := cfg.Set("port", p.server.Port()); err != nil {
		return errs.Wrap(err, "Could not save config")
	}

	// Handle sigterm
	p.sig = make(chan os.Signal, 1)
	signal.Notify(p.sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		oscall := <-p.sig
		logging.Debug("system call:%+v", oscall)
		p.Stop()
	}()

	if err := p.server.Start(); err != nil {
		return errs.Wrap(err, "Failed to start server")
	}

	return nil
}

func (p *program) Stop() error {
	if p.server == nil {
		return errs.New("Can't stop program as it was never started")
	}

	if err := p.server.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Closing server failed: %v", err)
	}
	signal.Stop(p.sig)
	return nil
}
