package runbits

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

// Invoke will invoke a state tool command with the given args and prints a friendly message indicating what we're doing
func Invoke(out output.Outputer, args ...string) error {
	// Tell user we're invoking a state command
	out.Notice(locale.Tl("tutorial_invoking", "\n[INFO]Invoking `state {{.V0}}` ...[/RESET]", strings.Join(args, " ")))
	time.Sleep(time.Second)

	// Get terminal width so we can print dashed line to call out state command output
	termWidth, _, err := terminal.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		logging.Debug("Cannot get terminal size: %v", err)
		termWidth = 100
	}

	// print dashed line
	out.Notice("[INFO]" + strings.Repeat("-", termWidth) + "[/RESET]")

	// Execute state command
	exe, err := os.Executable()
	if err != nil {
		return locale.WrapError(err, "err_tutorial_invoke_exe", "Could not detect executable path of State Tool.")
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err = cmd.Run()

	// print dashed line
	out.Notice("[INFO]" + strings.Repeat("-", termWidth) + "[/RESET]")

	if err != nil {
		return locale.WrapError(err, "err_tutorial_invoke_run", "Errors occurred while invoking State Tool command: `state {{.V0}}`.", strings.Join(args, " "))
	}

	return nil
}
