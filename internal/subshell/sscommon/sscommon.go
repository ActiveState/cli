package sscommon

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/sighandler"
)

func NewCommand(command string, args []string, env []string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	return cmd
}

// Start wires stdin/stdout/stderr into the provided command, starts it, and
// returns a channel to monitor errors on.
func Start(cmd *exec.Cmd) chan error {
	logging.Debug("Starting subshell with cmd: %s", cmd.String())

	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	cmd.Start()

	errors := make(chan error, 1)

	go func() {
		defer close(errors)

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

				errors <- errs.WrapExitCode(eerr, code, true)
				return
			}

			errors <- errs.Wrap(err, "Command Failed: %s", cmd.String())
			return
		}
	}()

	return errors
}

// Stop signals the provided command to terminate.
func Stop(cmd *exec.Cmd) error {
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
	filePath, err := osutils.BashifyPath(name)
	if err != nil {
		return err
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
		pythonPath, err := binaryPathCmd(env, "python")
		if err != nil {
			return err
		}
		name = pythonPath
	case ".pl":
		args = append([]string{name}, args...)
		perlPath, err := binaryPathCmd(env, "perl")
		if err != nil {
			return err
		}
		name = perlPath
	case ".bat":
		// No action required
	case ".ps1":
		args = append([]string{"-file", name}, args...)
		name = "powershell"
	case ".sh":
		bashPath, err := osutils.BashifyPath(name)
		if err != nil {
			return locale.WrapError(
				err, "err_sscommon_cannot_translate_path",
				"Cannot translate Windows path ({{.V0}}) to bash path.", name,
			)
		}
		args = append([]string{bashPath}, args...)
		name = "bash"
	default:
		return locale.NewInputError("err_sscommon_unsupported_language", "", ext)
	}

	return runDirect(env, name, args...)
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

	err := runCmd.Run()
	// silence exit code errors
	if eerr, ok := err.(*exec.ExitError); ok {
		return errs.WrapExitCode(eerr, eerr.ExitCode(), true)
	}
	return err
}
