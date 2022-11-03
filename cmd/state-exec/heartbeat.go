package main

import (
	"os"
	"strconv"

	"github.com/ActiveState/cli/internal/svcctl/svcmsg"
)

func newHeartbeat(execPath string) (*svcmsg.Heartbeat, error) {
	pid := strconv.Itoa(os.Getpid())
	hb := svcmsg.NewHeartbeat(pid, execPath)

	return hb, nil
}
