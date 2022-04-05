package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/ActiveState/cli/cmd/state-svc/internal/server"
	anaSvc "github.com/ActiveState/cli/internal/analytics/client/sync"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/svcctl"
)

type service struct {
	cfg      *config.Instance
	an       *anaSvc.Client
	shutdown context.CancelFunc
	server   *server.Server
	ipcSrv   *ipc.Server
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

	logging.Debug("Server starting on port: %d", s.server.Port())

	go func() {
		if err := s.server.Start(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				logging.Error("%s", errs.Wrap(err, "Failed to start server"))
			}
		}
	}()
	defer s.shutdown()

	spath := svcctl.NewIPCSockPathFromGlobals()
	reqHandlers := []ipc.RequestHandler{ // caller-defined handlers to expand ipc capabilities
		svcctl.HTTPAddrHandler(".:" + strconv.Itoa(s.server.Port())),
	}
	s.ipcSrv = ipc.NewServer(spath, reqHandlers...)
	err = s.ipcSrv.Start()
	if err != nil {
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

	if err := s.ipcSrv.Close(); err != nil {
		return errs.Wrap(err, "Failed to stop ipc server")
	}

	return nil
}
