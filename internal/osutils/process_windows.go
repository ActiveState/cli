// +build windows

package osutils

import "syscall"

// SysProcAttrForNewProcessGroup returns a SysProcAttr structure configured to start a process with a new process group
func SysProcAttrForNewProcessGroup() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func SysProcAttrForBackgroundProcess() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x08000000, // CREATE_NO_WINDOW
	}
}
