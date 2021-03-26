package main

import (
	serviceProvider "github.com/kardianos/service"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

type Service struct {
	provider serviceProvider.Service
}

func NewService(handler *svcHandler) (*Service, error) {
	sp, err := serviceProvider.New(handler, &serviceProvider.Config{
		Name:        "StateSvc",
		DisplayName: "State Tool Service",
		Description: "The State Tool service supplies persistent information for use with the State Tool and ActiveState Desktop.",
	})

	if err != nil {
		return nil, errs.Wrap(err, "Could not create service")
	}

	return &Service{sp}, nil
}

func (s *Service) Start() error {
	status, err := s.provider.Status()
	if err != nil {
		return errs.Wrap(err, "Could not get service status")
	}

	if status == serviceProvider.StatusRunning {
		logging.Debug("Service is already started")
		return nil
	}

	logging.Debug("Invoking provider:Start")
	if err := s.provider.Start(); err != nil {
		return errs.Wrap(err, "Could not start service")
	}

	return nil
}

func (s *Service) Stop() error {
	status, err := s.provider.Status()
	if err != nil {
		return errs.Wrap(err, "Could not get service status")
	}

	if status == serviceProvider.StatusStopped {
		logging.Debug("Service is already stopped")
		return nil
	}

	if err := s.Stop(); err != nil {
		return errs.Wrap(err, "Could not stop service")
	}

	return nil
}

func (s *Service) IsInstalled() (bool, error) {
	_, err := s.provider.Status()
	if err != nil {
		if err == serviceProvider.ErrNotInstalled {
			return false, nil
		}
		return false, errs.Wrap(err, "Could not detect service status")
	}
	return true, nil
}

func (s *Service) Install() error {
	return s.provider.Install()
}

func (s *Service) Uninstall() error {
	return s.provider.Uninstall()
}
