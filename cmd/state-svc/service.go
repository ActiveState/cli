package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/ActiveState/cli/cmd/state-svc/internal/server"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

type service struct {
	cfg    *config.Instance
	ctx    context.Context
	server *server.Server
}

func NewService(cfg *config.Instance, ctx context.Context) *service {
	return &service{cfg: cfg, ctx: ctx}
}

func (s *service) Start() error {
	logging.Debug("service:Start")

	var err error
	s.server, err = server.New(s.cfg, s.ctx)
	if err != nil {
		return errs.Wrap(err, "Could not create server")
	}

	if err := s.cfg.Set(constants.SvcConfigPort, s.server.Port()); err != nil {
		return errs.Wrap(err, "Could not save config")
	}

	logging.Debug("Server starting on port: %d", s.server.Port())

	if err := s.server.Start(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return errs.Wrap(err, "Failed to start server")
	}

	return nil
}

func (s *service) Wait() error {
	if s.server == nil {
		return errs.New("Can't wait for service as it was never started")
	}

	s.server.Wait()
	return nil
}
