package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/shirou/gopsutil/process"
	"github.com/spf13/cast"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/ActiveState/cli/pkg/platform/api/svc"

	configMediator "github.com/ActiveState/cli/internal/mediators/config"
)

var ErrSvcAlreadyRunning error = errs.New("Service is already running")

func init() {
	configMediator.NewRule(constants.SvcConfigPid, configMediator.Int, configMediator.EmptyEvent, configMediator.EmptyEvent)
	configMediator.NewRule(constants.SvcConfigPort, configMediator.Int, configMediator.EmptyEvent, configMediator.EmptyEvent)
}

type serviceManager struct {
	cfg *config.Instance
}

func NewServiceManager(cfg *config.Instance) *serviceManager {
	return &serviceManager{cfg}
}

func (s *serviceManager) Start(args ...string) error {
	var proc *os.Process
	err := s.cfg.GetThenSet(constants.SvcConfigPid, func(currentValue interface{}) (interface{}, error) {
		oldPid := cast.ToInt(currentValue)
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
			err := rtutils.Timeout(func() error { return proc.Signal(os.Kill) }, time.Second)
			if err != nil {
				logging.Errorf("Could not cleanup process: %v", err)
				fmt.Printf("Failed to cleanup serviceManager after lock failed, please manually kill the following pid: %d\n", proc.Pid)
			}
		}
		return err
	}

	logging.Debug("Process started using pid %d", proc.Pid)
	return nil
}

func (s *serviceManager) Stop() error {
	pid, err := s.CheckPid(s.cfg.GetInt(constants.SvcConfigPid))
	if err != nil {
		return errs.Wrap(err, "Could not get pid")
	}
	if pid == nil {
		logging.Debug("State service is not running. Nothing to stop")
		return nil
	}

	if err := stopServer(s.cfg); err != nil {
		return errs.Wrap(err, "Failed to stop server")
	}

	return nil
}

func stopServer(cfg *config.Instance) error {
	htClient := retryhttp.DefaultClient.StandardClient()

	client, err := svc.New(cfg)
	if err != nil {
		return errs.Wrap(err, "Could not initialize svc client")
	}

	quitAddress := fmt.Sprintf("%s/__quit", client.BaseUrl())
	logging.Debug("Sending quit request to %s", quitAddress)
	req, err := http.NewRequest("GET", quitAddress, nil)
	if err != nil {
		return errs.Wrap(err, "Could not create request to quit svc")
	}

	res, err := htClient.Do(req)
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

// CheckPid checks if the given pid refers to an existing process
func (s *serviceManager) CheckPid(pid int) (*int, error) {
	if pid == 0 {
		return nil, nil
	}
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return nil, nil
		}
		return nil, errs.Wrap(err, "Could not verify if pid exists")
	}

	// Try to verify that the matching pid is actually our process, because Windows aggressively reuses PIDs
	if runtime.GOOS == "windows" {
		exe, err := p.Exe()
		if err != nil {
			logging.Error("Could not detect executable for pid, error: %s", errs.JoinMessage(err))
		} else if !strings.HasPrefix(strings.ToLower(filepath.Base(exe)), constants.ServiceCommandName) {
			return nil, nil
		}
	}

	var rpid = int(p.Pid)
	return &rpid, nil
}
