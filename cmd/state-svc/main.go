package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

type command string

const (
	CmdBackground = "background"
)

var commands = []command{
	CmdBackground,
}

func main() {
	err := run()
	if err != nil {
		errMsg := errs.Join(err, ": ").Error()
		logging.Errorf("state-svc errored out: %s", errMsg)
		fmt.Fprintln(os.Stderr, errMsg)
		os.Exit(1)
	}
}

func run() error {
	cmd := command(os.Args[1])

	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config")
	}

	switch cmd {
	case CmdBackground:
		logging.Debug("Running CmdBackground")
		return runBackground(cfg)
	}

	return runForeground(cfg)
}

func runForeground(cfg *config.Instance) error {
	logging.Debug("Running in foreground")

	p := NewProgram(cfg)
	if err := p.Start(); err != nil {
		return errs.Wrap(err, "Could not start program")
	}

	return nil
}

func runBackground(cfg *config.Instance) error {
	logging.Debug("Running in background")

	// Don't run in background if we're already running
	port := cfg.GetInt("port")
	if port > 0 {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port)), time.Second)
		if err == nil && conn != nil {
			conn.Close()
			return errs.New("Service is already running on port %d", port)
		}
	}
	
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	return cmd.Start()
}
