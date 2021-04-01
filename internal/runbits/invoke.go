package runbits

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/termutils"
)

// Invoke will invoke a state tool command with the given args and prints a friendly message indicating what we're doing
func Invoke(out output.Outputer, args ...string) error {
	// Tell user we're invoking a state command
	out.Notice(locale.Tl("tutorial_invoking", "\n[NOTICE]Invoking `state {{.V0}}` ...[/RESET]", strings.Join(args, " ")))
	time.Sleep(time.Second)

	// Get terminal width so we can print dashed line to call out state command output
	termWidth := termutils.GetWidth()

	// print dashed line
	out.Notice("[NOTICE]" + strings.Repeat("-", termWidth) + "[/RESET]")

	err := InvokeSilent(args...)

	// print dashed line
	out.Notice("[NOTICE]" + strings.Repeat("-", termWidth) + "[/RESET]\n")

	if err != nil {
		return err
	}

	return nil
}

// InvokeSilent just invokes a given state tool command in the background, silently from the user
func InvokeSilent(args ...string) error {
	// Execute state command
	exe, err := os.Executable()
	if err != nil {
		return locale.WrapError(err, "err_tutorial_invoke_exe", "Could not detect executable path of State Tool.")
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err = cmd.Run()

	if err != nil {
		return locale.WrapInputError(err, "err_tutorial_invoke_run", "Errors occurred while invoking State Tool command: `state {{.V0}}`.", strings.Join(args, " "))
	}

	return nil
}
