package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ActiveState/cli/cmd/state-exec/internal/logr"
)

const (
	executorName     = "state-exec"
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
		fmt.Fprintf(os.Stderr, "[DEBUG %9d] ", time.Since(start).Nanoseconds())
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

func main() {
	runtime.GOMAXPROCS(1)

	if os.Getenv(envVarKeyVerbose) == "true" {
		logr.SetDebug(logDbgFunc(time.Now()))
	}

	if err := run(); err != nil {
		logErr(userErrMsg)
		logErr("%s", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run() error {
	logr.Debug("run hello")
	defer logr.Debug("run goodbye")

	hb, err := newHeartbeat()
	if err != nil {
		return err
	}
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

	return runCmd(meta)
}
