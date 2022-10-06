package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ActiveState/cli/internal/svcctl/svcmsg"
)

func newHeartbeat() (*svcmsg.Heartbeat, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("new heartbeat: %w", err)
	}
	pid := strconv.Itoa(os.Getpid())
	hb := svcmsg.NewHeartbeat(pid, execPath)

	return hb, nil
}
