package sscommon

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/sighandler"
)

var (
	// FailSignalCmd represents a failure sending a system signal to a cmd
	FailSignalCmd = failures.Type("sscommon.fail.signalcmd")
)

type silentExitCodeError struct {
	error
}

func (se silentExitCodeError) Unwrap() error {
	return se.error
}

// IsSilent returns true, as no State Tool error message should be written for errors caused inside the sub-shell
func (se silentExitCodeError) IsSilent() bool {
	return true
}

// Start wires stdin/stdout/stderr into the provided command, starts it, and
// returns a channel to monitor errors on.
func Start(cmd *exec.Cmd) chan error {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	cmd.Start()

	errs := make(chan error, 1)

	go func() {
		defer close(errs)

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

				errs <- silentExitCodeError{eerr}
				return
			}

			errs <- err
			return
		}
	}()

	return errs
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
		linPath, err := winPathToLinPath(name)
		if err != nil {
			return locale.WrapError(
				err, "err_sscommon_cannot_translate_path",
				"Cannot translate Windows path ({{.V0}}) to bash path.", name,
			)
		}
		args = append([]string{linPath}, args...)
		name = "bash"
	default:
		return failures.FailUser.New("err_sscommon_unsupported_language", ext)
	}

	return runDirect(env, name, args...)
}

func winPathToLinPath(name string) (string, error) {
	cmd := exec.Command("bash", "-c", "pwd")
	cmd.Dir = filepath.Dir(name)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	path := strings.TrimSpace(string(out)) + "/" + filepath.Base(name)

	return path, nil
}

func binaryPathCmd(env []string, name string) (string, error) {
	cmd := exec.Command("where", name)
	cmd.Env = env

	out, err := cmd.Output()
	if err != nil {
		return "", errs.Wrap(err, "Failed to get output of %s", strings.Join(cmd.Args, " "))
	}

	split := strings.Split(string(out), "\r\n")
	if len(split) == 0 {
		return "", locale.NewInputError("err_sscommon_binary_path", name)
	}

	return split[0], nil
}

func runDirect(env []string, name string, args ...string) error {
	logging.Debug("Running command: %s %s", name, strings.Join(args, " "))

	runCmd := exec.Command(name, args...)
	runCmd.Stdin, runCmd.Stdout, runCmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	runCmd.Env = env

	// CTRL+C interrupts are sent to all processes in a terminal at the same
	// time (with some extra control through process groups).
	// Here is what can happen *without* the next line:
	// - `state run` gets interrupted and exits, returning to the parent shell.
	// - child processes started by state run ignores or handles interrupt, and stays alive.
	// - the parent shell and the child process read from stdin simultaneously.
	// This behavior has been reported in
	// - https://www.pivotaltracker.com/story/show/169509213 and
	// - https://www.pivotaltracker.com/story/show/167523128
	bs := sighandler.NewBackgroundSignalHandler(func(_ os.Signal) {}, os.Interrupt)
	sighandler.Push(bs)
	defer sighandler.Pop()

	return runCmd.Run()
}
