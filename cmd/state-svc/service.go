package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/ActiveState/cli/cmd/state-svc/internal/server"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

type service struct {
	cfg    *config.Instance
	server *server.Server
}

func NewService(cfg *config.Instance) *service {
	return &service{cfg: cfg}
}

func (s *service) Start() error {
	logging.Debug("service:Start")

	var err error
	s.server, err = server.New(s.cfg)
	if err != nil {
		return errs.Wrap(err, "Could not create server")
	}

	if err := s.cfg.Set(constants.SvcConfigPort, s.server.Port()); err != nil {
		return errs.Wrap(err, "Could not save config")
	}

	if err := s.server.Start(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return errs.Wrap(err, "Failed to start server")
	}

	return nil
}

func (s *service) Stop() error {
	if s.server == nil {
		return errs.New("Can't stop service as it was never started")
	}

	if err := s.server.Shutdown(); err != nil {
		fmt.Fprintf(os.Stderr, "Closing server failed: %v", err)
	}
	return nil
}
