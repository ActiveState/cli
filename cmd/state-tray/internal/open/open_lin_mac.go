//+build !windows

package open

import (
	"fmt"
	"os/exec"

	"github.com/ActiveState/cli/internal/locale"
)

func Prompt(command string) error {
	script := fmt.Sprintf(`tell application "Terminal" to do script "%s"`, command)
	cmd := exec.Command(`osascript`, "-s", "h", "-e", script)
	err := cmd.Run()
	if err != nil {
		return locale.WrapError(err, "err_open_prompt", "Could not open prompt")
	}

	return nil
}
