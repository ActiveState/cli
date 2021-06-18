package main

import (
	"context"
	"fmt"
	"os"

	"github.com/shirou/gopsutil/process"
	"github.com/spf13/cast"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/platform/model"
)

var ErrSvcAlreadyRunning error = errs.New("Service is already running")

type serviceManager struct {
	cfg *config.Instance
}

func NewServiceManager(cfg *config.Instance) *serviceManager {
	return &serviceManager{cfg}
}

func (s *serviceManager) Start(args ...string) error {
	var proc *os.Process
	err := s.cfg.SetWithLock(constants.SvcConfigPid, func(oldPidI interface{}) (interface{}, error) {
		oldPid := cast.ToInt(oldPidI)
		curPid, err := s.CheckPid(oldPid)
		if err == nil && curPid != nil {
			return nil, ErrSvcAlreadyRunning
		}

		proc, err = exeutils.ExecuteAndForget(args[0], args[1:])
		if err != nil {
			return nil, errs.New("Could not start serviceManager")
		}

		if proc == nil {
			return nil, errs.New("Could not obtain process information after starting serviceManager")
		}

		return proc.Pid, nil
	})
	if err != nil {
		if proc != nil {
			if err := proc.Signal(os.Interrupt); err != nil {
				logging.Errorf("Could not cleanup process: %v", err)
				fmt.Printf("Failed to cleanup serviceManager after lock failed, please manually kill the following pid: %d\n", proc.Pid)
			}
		}
		return errs.Wrap(err, "Could not store pid")
	}

	logging.Debug("Process started using pid %d", proc.Pid)
	return nil
}

func (s *serviceManager) Stop() error {
	err := s.cfg.Reload()
	if err != nil {
		return errs.Wrap(err, "Failed to reload configuration")
	}
	pid, err := s.CheckPid(s.cfg.GetInt(constants.SvcConfigPid))
	if err != nil {
		return errs.Wrap(err, "Could not get pid")
	}
	if pid == nil {
		return nil
	}

	// Ensure that port number has been written to configuration file ie., that the server is ready to talk
	svcmgr := svcmanager.New(s.cfg)

	ctx, cancel := context.WithTimeout(context.Background(), svcmanager.MinimalTimeout)
	defer cancel()
	svcm, err := model.NewSvcModel(ctx, s.cfg, svcmgr)
	if err != nil {
		return errs.Wrap(err, "Could not initialize svc model")
	}

	if err := svcm.StopServer(); err != nil {
		return errs.Wrap(err, "Failed to stop server")
	}
	return nil
}

// CheckPid checks if the given pid revers to an existing process
func (s *serviceManager) CheckPid(pid int) (*int, error) {
	if pid == 0 {
		return nil, nil
	}
	pidExists, err := process.PidExists(int32(pid))
	if err != nil {
		return nil, errs.Wrap(err, "Could not verify if pid exists")
	}
	if !pidExists {
		return nil, nil
	}

	return &pid, nil
}
