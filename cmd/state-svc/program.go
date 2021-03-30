package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ActiveState/cli/cmd/state-svc/internal/server"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

type program struct {
	cfg    *config.Instance
	server *server.Server
}

func NewProgram(cfg *config.Instance) *program {
	return &program{cfg: cfg}
}

func (p *program) Start() error {
	logging.Debug("program:Start")

	var err error
	p.server, err = server.New()
	if err != nil {
		return errs.Wrap(err, "Could not create server")
	}

	if err := p.cfg.Set(constants.SvcConfigPort, p.server.Port()); err != nil {
		return errs.Wrap(err, "Could not save config")
	}

	// Handle sigterm
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	go func() {
		oscall := <-sig
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

	if err := p.server.Shutdown(); err != nil {
		fmt.Fprintf(os.Stderr, "Closing server failed: %v", err)
	}
	return nil
}
