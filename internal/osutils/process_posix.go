// +build linux darwin

package osutils

import "syscall"

// SysProcAttrForNewProcessGroup returns a SysProcAttr structure configured to start a process with a new process group
func SysProcAttrForNewProcessGroup() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}

func SysProcAttrForBackgroundProcess() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}
