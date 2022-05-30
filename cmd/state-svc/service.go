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
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type service struct {
	ctx    context.Context
	cfg    *config.Instance
	an     *anaSvc.Client
	auth   *authentication.Auth
	server *server.Server
	ipcSrv *ipc.Server
}

func NewService(ctx context.Context, cfg *config.Instance, an *anaSvc.Client, auth *authentication.Auth) *service {
	return &service{ctx: ctx, cfg: cfg, an: an, auth: auth}
}

func (s *service) Start() error {
	logging.Debug("service:Start")

	var err error
	s.server, err = server.New(s.cfg, s.an, s.auth)
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

	spath := svcctl.NewIPCSockPathFromGlobals()
	reqHandlers := []ipc.RequestHandler{ // caller-defined handlers to expand ipc capabilities
		svcctl.HTTPAddrHandler(":" + strconv.Itoa(s.server.Port())),
		svcctl.LogFileHandler(logging.FileName()),
	}
	s.ipcSrv = ipc.NewServer(s.ctx, spath, reqHandlers...)
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

	s.ipcSrv.Shutdown()

	return nil
}

func (s *service) Wait() error {
	if err := s.ipcSrv.Wait(); err != nil {
		return errs.Wrap(err, "IPC server operating failure")
	}
	return nil
}
