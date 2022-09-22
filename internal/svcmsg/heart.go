package svcmsg

import (
	"fmt"
	"strings"
)

type Heartbeat struct {
	ProcessID string
	ExecPath  string
}

func NewHeartbeatFromSvcMsg(data string) *Heartbeat {
	var pid, execPath string

	ss := strings.SplitN(data, "<", 2)
	if len(ss) > 0 {
		pid = ss[0]
	}
	if len(ss) > 1 {
		execPath = ss[1]
	}

	return NewHeartbeat(pid, execPath)
}

func NewHeartbeat(pid, execPath string) *Heartbeat {
	return &Heartbeat{
		ProcessID: pid,
		ExecPath:  execPath,
	}
}

func (h *Heartbeat) SvcMsg() string {
	return fmt.Sprintf("heart<%s<%s", h.ProcessID, h.ExecPath)
}
