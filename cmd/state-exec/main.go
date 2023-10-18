package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
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

func init() {
	// This application is not doing enough to warrant parallelism, so let's
	// skip it and avoid the cost of scheduling.
	runtime.GOMAXPROCS(1)
}

func main() {
	if os.Getenv(envVarKeyVerbose) == "true" {
		logr.SetDebug(logDbgFunc(time.Now()))
	}

	if err := run(); err != nil {
		if exitErr := (&exec.ExitError{}); errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}

		logErr("run failed: %s", err)
		logErr(userErrMsg)
		os.Exit(1)
	}

	os.Exit(0)
}

func run() error {
	logr.Debug("hello")
	defer logr.Debug("run goodbye")

	hb, err := newHeartbeat()
	if err != nil {
		return fmt.Errorf("cannot create new heartbeat: %w", err)
	}
	logr.Debug("message data - pid: %s, exec: %s", hb.ProcessID, hb.ExecPath)

	meta, err := newExecutorMeta(hb.ExecPath)
	if err != nil {
		return fmt.Errorf("cannot create new executor meta: %w", err)
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
		logr.Debug("                 sock - error: %v", err)

		if inActiveStateCI() { // halt control flow on CI only
			return fmt.Errorf("cannot send message to service (this error is handled in CI only): %w", err)
		}
	}

	logr.Debug("cmd - running: %s", meta.MatchingBin)
	exitCode, err := runCmd(meta)
	if err != nil {
		logr.Debug("      running - failed: bins (%v)", meta.ExecMeta.Bins)
		return fmt.Errorf("cannot run command: %w", err)
	}

	msg, err := newExitCodeMessage(exitCode)
	if err != nil {
		return fmt.Errorf("cannot create new exit code message: %w", err)
	}
	logr.Debug("message data - exec: %s, exit code: %s", msg.ExecPath, msg.ExitCode)

	if err := sendMsgToService(meta.SockPath, msg); err != nil {
		logr.Debug("                 sock - error: %v", err)

		if inActiveStateCI() { // halt control flow on CI only
			return fmt.Errorf("cannot send message to service (this error is handled in CI only): %w", err)
		}
	}

	return nil
}
