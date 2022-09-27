package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ActiveState/cli/cmd/state-execx/internal/logr"
	"github.com/ActiveState/cli/internal/svcmsg"
)

const (
	executorName     = "state-execx"
	envVarKeyVerbose = "ACTIVESTATE_VERBOSE"
	userErrMsg       = "Not user serviceable; Please contact support for assistance."
)

var (
	logErr = func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, "%s: ", executorName)
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
)

func logDbgFunc(start time.Time) logr.LogFunc {
	return func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, "[%12s %9d] ", executorName, time.Since(start).Nanoseconds())
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

func main() {
	runtime.GOMAXPROCS(1)

	if os.Getenv(envVarKeyVerbose) == "true" {
		logr.SetDebug(logDbgFunc(time.Now()))
	}

	if err := run(); err != nil {
		// TODO: do not log errors if exiterror, just exit non-zero
		logErr(userErrMsg)
		logErr("%s", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run() error {
	logr.Debug("run hello")
	defer logr.Debug("run goodbye")

	hb := svcmsg.NewHeartbeat(os.Args[2], os.Args[1])
	logr.Debug("message data - pid: %s, exec: %s", hb.ProcessID, hb.ExecPath)

	meta, err := newExecutorMeta(hb.ExecPath)
	if err != nil {
		return err
	}
	logr.CallIfDebugIsSet(func() {
		logr.Debug("meta data - bins...")
		for _, bin := range meta.Bins {
			logr.Debug("            bins : %s", bin)
		}
	})
	logr.Debug("meta data - matching bin: %s", meta.MatchingBin)
	logr.CallIfDebugIsSet(func() {
		logr.Debug("meta data - env...")
		for _, entry := range meta.TransformedEnv {
			logr.Debug("            env - kv: %s", entry)
		}
	})

	logr.Debug("communications - sock: %s", meta.SockPath)
	if err := sendMsgToService(meta.SockPath, hb); err != nil {
		return err
	}

	return runCmd(meta, os.Args[3:])
}
