//+build linux

package open

import (
	"fmt"
	"os/exec"

	"github.com/ActiveState/cli/internal/locale"
)

func Prompt(command string) error {
	shellData, err := exec.Command("bash", "-c",
		"grep ^$(id -un): /etc/passwd | cut -d: -f7-",
	).Output()
	if err != nil {
		return locale.WrapError(err,
			"err_determine_default_shell", "Could not determine default shell",
		)
	}
	shell := string(shellData[:len(shellData)-1]) // trim newline

	command = fmt.Sprintf("%s;%s", command, shell)
	cmd := exec.Command("x-terminal-emulator", "-e", shell, "-c", command)
	if err := cmd.Run(); err != nil {
		return locale.WrapError(err, "err_open_prompt", "Could not open prompt")
	}

	return nil
}
