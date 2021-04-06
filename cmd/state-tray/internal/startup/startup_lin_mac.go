//+build !windows

package startup

import (
	"os/exec"
)

func newStartServiceCommand(path string) *exec.Cmd {
	return exec.Command(path, "start")
}
