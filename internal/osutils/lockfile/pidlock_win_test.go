// +build windows

package lockfile

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
)

func sendCtrlBreak(t *testing.T, pid int) {
	d, e := syscall.LoadDLL("kernel32.dll")
	if e != nil {
		t.Fatalf("LoadDLL: %v\n", e)
	}
	p, e := d.FindProc("GenerateConsoleCtrlEvent")
	if e != nil {
		t.Fatalf("FindProc: %v\n", e)
	}
	r, _, e := p.Call(syscall.CTRL_BREAK_EVENT, uintptr(pid))
	if r == 0 {
		t.Fatalf("GenerateConsoleCtrlEvent: %v\n", e)
	}
}

func interruptProcess(t *testing.T, p *os.Process) {
	sendCtrlBreak(t, p.Pid)
}

func prepLockCmd(lockCmd *exec.Cmd) *exec.Cmd {
	lockCmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	return lockCmd
}
