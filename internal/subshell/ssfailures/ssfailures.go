package ssfailures

import "github.com/ActiveState/cli/internal/failures"

var (
	// FailExecCmd represents a failure running a cmd
	FailExecCmd = failures.Type("ssfailures.fail.execcmd")

	// FailSignalCmd represents a failure sending a system signal to a cmd
	FailSignalCmd = failures.Type("ssfailures.fail.signalcmd")
)
