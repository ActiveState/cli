// Package svcmsg models the Exit Code data that the executor must communicate
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

type ExitCode struct {
	ExecPath string
	ExitCode string
}

func NewExitCodeFromSvcMsg(data string) *ExitCode {
	var execPath, exitCode string

	ss := strings.SplitN(data, "<", 2)
	if len(ss) > 0 {
		execPath = ss[0]
	}
	if len(ss) > 1 {
		exitCode = ss[1]
	}

	return NewExitCode(execPath, exitCode)
}

func NewExitCode(execPath, exitCode string) *ExitCode {
	return &ExitCode{
		ExecPath: execPath,
		ExitCode: exitCode,
	}
}

func (e *ExitCode) SvcMsg() string {
	return fmt.Sprintf("exitcode<%s<%s", e.ExecPath, e.ExitCode)
}
