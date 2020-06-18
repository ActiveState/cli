package sscommon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
)

var (
	// FailExecCmd represents a failure running a cmd
	FailExecCmd = failures.Type("sscommon.fail.execcmd")

	// FailExecCmdExit represents a cmd exit error failure
	FailExecCmdExit = failures.Type("sscommon.fail.execcmdexit", failures.FailSilent)

	// FailSignalCmd represents a failure sending a system signal to a cmd
	FailSignalCmd = failures.Type("sscommon.fail.signalcmd")
)

// Start wires stdin/stdout/stderr into the provided command, starts it, and
// returns a channel to monitor errors on.
func Start(cmd *exec.Cmd) chan *failures.Failure {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	cmd.Start()

	fs := make(chan *failures.Failure, 1)

	go func() {
		defer close(fs)

		if err := cmd.Wait(); err != nil {
			if eerr, ok := err.(*exec.ExitError); ok {
				code := eerr.ExitCode()
				valid := eerr.Exited()
				// code 130 is returned when a process halts
				// due to SIGTERM after receiving a SIGINT
				// code -1 is returned when a process halts
				// due to SIGTERM without any interference.
				if code == 130 || (valid && code == -1) {
					logging.Debug("exit - valid: %t, code: %d", valid, code)
					return
				}

				fs <- FailExecCmdExit.Wrap(eerr)
				return
			}

			fs <- FailExecCmd.Wrap(err)
			return
		}
	}()

	return fs
}

// Stop signals the provided command to terminate.
func Stop(cmd *exec.Cmd) *failures.Failure {
	return stop(cmd)
}

// RunFunc ...
type RunFunc func(env []string, name string, args ...string) error

func RunFuncByBinary(binary string) RunFunc {
	bin := strings.ToLower(binary)
	switch {
	case strings.Contains(bin, "bash"):
		return runWithBash
	case strings.Contains(bin, "cmd"):
		return runWithCmd
	default:
		return runDirect
	}
}

func runWithBash(env []string, name string, args ...string) error {
	filePath, fail := osutils.BashifyPath(name)
	if fail != nil {
		return fail.ToError()
	}

	esc := osutils.NewBashEscaper()

	quotedArgs := filePath
	for _, arg := range args {
		quotedArgs += " " + esc.Quote(arg)
	}

	return runDirect(env, "bash", "-c", quotedArgs)
}

func runWithCmd(env []string, name string, args ...string) error {
	ext := filepath.Ext(name)
	switch ext {
	case ".py":
		args = append([]string{name}, args...)
		pythonPath, fail := binaryPathCmd(env, "python")
		if fail != nil {
			return fail
		}
		name = pythonPath
	case ".pl":
		args = append([]string{name}, args...)
		perlPath, fail := binaryPathCmd(env, "perl")
		if fail != nil {
			return fail
		}
		name = perlPath
	case ".bat":
		// No action required
	case ".ps1":
		args = append([]string{"-file", name}, args...)
		name = "powershell"
	case ".sh":
		name = winPathToLinPath(name)
		args = append([]string{"-c", name}, args...)
		name = "bash"
	default:
		return failures.FailUser.New("err_sscommon_unsupported_language", ext)
	}

	return runDirect(env, name, args...)
}

func winPathToLinPath(name string) string {
	if true {
		return name
	}
	ss := strings.SplitN(name, ":", 2)
	if len(ss) < 2 {
		return name
	}

	drive := strings.ToLower(ss[0])
	path := strings.ReplaceAll(ss[1], string(os.PathSeparator), "/")

	return fmt.Sprintf("/mnt/%s%s", drive, path)
}

func binaryPathCmd(env []string, name string) (string, error) {
	cmd := exec.Command("where", "python")
	cmd.Env = env

	out, err := cmd.Output()
	if err != nil {
		return "", FailExecCmd.Wrap(err)
	}

	split := strings.Split(string(out), "\r\n")
	if len(split) == 0 {
		return "", failures.FailCmd.New("err_sscommon_binary_path", name)
	}

	return split[0], nil
}

func ignoreInterrupts(ctx context.Context) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT)
	go func() {
		defer close(c)
		defer signal.Stop(c)
		for {
			select {
			case <-c:
				logging.Debug("Received a SIGINT interrupt")
			case <-ctx.Done():
				return
			}
		}
	}()
}

func runDirect(env []string, name string, args ...string) error {
	logging.Debug("Running command: %s %s", name, strings.Join(args, " "))

	runCmd := exec.Command(name, args...)
	runCmd.Stdin, runCmd.Stdout, runCmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	runCmd.Env = env

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// CTRL+C interrupts are sent to all processes in a terminal at the same
	// time (with some extra control through process groups).
	// Here is what can happen *without* the next line:
	// - `state run` gets interrupted and exits, returning to the parent shell.
	// - child processes started by state run ignores or handles interrupt, and stays alive.
	// - the parent shell and the child process read from stdin simultaneously.
	// This behavior has been reported in
	// - https://www.pivotaltracker.com/story/show/169509213 and
	// - https://www.pivotaltracker.com/story/show/167523128
	ignoreInterrupts(ctx)

	return runCmd.Run()
}
