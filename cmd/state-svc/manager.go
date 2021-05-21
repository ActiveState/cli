package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/shirou/gopsutil/process"
	"github.com/spf13/cast"

	"github.com/ActiveState/cli/cmd/state-svc/internal/server"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/svcmanager"
)

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
			return nil, errs.New("Service is already running")
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
	waitMgr := svcmanager.New(s.cfg)
	err = waitMgr.Wait()
	if err != nil {
		if proc, perr := process.NewProcess(int32(*pid)); perr == nil {
			proc.Kill()
		}
		return errs.Wrap(err, "Failed to wait for background service to become responsive for clean shutdown, sent KILL signal")
	}

	port := s.cfg.GetInt(constants.SvcConfigPort)
	quitAddress := fmt.Sprintf("http://127.0.0.1:%d%s", port, server.QuitRoute)
	logging.Debug("Sending quit request to %s", quitAddress)
	req, err := http.NewRequest("GET", quitAddress, nil)
	if err != nil {
		return errs.Wrap(err, "Could not create request to quit svc")
	}

	client := &http.Client{
		Timeout: time.Second * 120,
	}
	res, err := client.Do(req)
	if err != nil {
		return errs.Wrap(err, "Request to quit svc failed")
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return errs.Wrap(err, "Request to quit svc responded with status %s", res.Status)
		}
		return errs.New("Request to quit svc responded with status: %s, response: %s", res.Status, body)
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
