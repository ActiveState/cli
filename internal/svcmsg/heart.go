// Package svcmsg models the Heartbeat data that the executor must communicate
// to the service.
//
// IMPORTANT: This package should have minimal dependencies as it will be
// imported by cmd/state-exec. The resulting compiled executable must remain as
// small as possible.
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
