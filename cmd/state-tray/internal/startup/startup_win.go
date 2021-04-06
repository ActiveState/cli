//+build windows

package startup

import (
	"os/exec"
	"syscall"
)

func newStartServiceCommand(path string) *exec.Cmd {
	cmd := exec.Command(path, "start")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return cmd
}
