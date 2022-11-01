package svcmsg

import (
	"fmt"
	"strings"
)

type Attempt struct {
	ExecPath string
}

func NewAttemptFromSvcMsg(data string) *Attempt {
	var execPath string

	ss := strings.SplitN(data, "<", 1)
	if len(ss) > 0 {
		execPath = ss[0]
	}

	return NewAttempt(execPath)
}

func NewAttempt(execPath string) *Attempt {
	return &Attempt{
		ExecPath: execPath,
	}
}

func (a *Attempt) SvcMsg() string {
	return fmt.Sprintf("attempt<%s", a.ExecPath)
}
