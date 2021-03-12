//+build windows

package open

import (
	"os/exec"

	"github.com/ActiveState/cli/internal/locale"
)

// Prompt will open the a command prompt and execute the given
// command string
func Prompt(command string) error {
	cmd := exec.Command("cmd.exe", "/c", "start", "", "cmd", "/k", command)
	err := cmd.Run()
	if err != nil {
		return locale.WrapError(err, "err_open_prompt", "Could not open prompt")
	}

	return nil
}
