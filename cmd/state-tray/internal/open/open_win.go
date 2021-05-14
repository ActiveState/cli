// +build windows

package open

import (
	"os/exec"

	"github.com/ActiveState/cli/internal/locale"
)

func TerminalAndWait(command string) error {
	return Terminal(command)
}

// Terminal will open the command prompt and execute the given command string
func Terminal(command string) error {
	// start will open an instance of the given executable. The first parameter
	// of start is the title, the second is the executable to start.
	cmd := exec.Command("cmd.exe", "/c", "start", "", "cmd.exe", "/c", command+" && pause")
	err := cmd.Run()
	if err != nil {
		return locale.WrapError(err, "err_open_prompt", "Could not open prompt")
	}

	return nil
}
