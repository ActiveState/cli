package main

import (
	"github.com/kardianos/service"

	"github.com/ActiveState/cli/internal/logging"
)

type svcHandler struct {
	program *program
}

func NewServiceHandler(p *program) *svcHandler {
	return &svcHandler{p}
}

func (s *svcHandler) Start(wrapperSvc service.Service) error {
	logging.Debug("svcHandler:Start")
	// Start should not block, according to the service package example code
	// Why don't they handle this package side? Good question ..
	go func() {
		if err := s.program.Start(); err != nil {
			logging.Errorf("Service encountered error: %v", err)
		}
	}()
	return nil
}

func (s *svcHandler) Stop(wrapperSvc service.Service) error {
	logging.Debug("svcHandler:Stop")
	return s.program.Stop()
}
