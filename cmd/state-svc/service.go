package main

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/ActiveState/cli/cmd/state-svc/internal/server"
	anaSvc "github.com/ActiveState/cli/internal/analytics/client/sync"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/spf13/cast"
)

type service struct {
	cfg      *config.Instance
	an       *anaSvc.Client
	shutdown context.CancelFunc
	server   *server.Server
}

func NewService(cfg *config.Instance, an *anaSvc.Client, shutdown context.CancelFunc) *service {
	return &service{cfg: cfg, an: an, shutdown: shutdown}
}

func (s *service) Start() error {
	logging.Debug("service:Start")

	var err error
	s.server, err = server.New(s.cfg, s.an, s.shutdown)
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

func (s *service) Stop() error {
	if s.server == nil {
		return errs.New("Can't stop service as it was never started")
	}

	if err := s.server.Shutdown(); err != nil {
		return errs.Wrap(err, "Failed to stop server")
	}

	err := s.cfg.GetThenSet(constants.SvcConfigPid, func(currentValue interface{}) (interface{}, error) {
		setPid := cast.ToInt(currentValue)
		if setPid != os.Getpid() {
			logging.Warning("PID in configuration file does not match PID of server shutting down")
			return config.CancelSet, nil
		}
		return "", nil
	})
	if err != nil {
		logging.Warning("Could not unset State Service PID in configuration file")
	}

	return nil
}
