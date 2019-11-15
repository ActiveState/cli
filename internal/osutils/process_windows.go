// +build windows

package osutils

import "syscall"

// SysProcAttrForNewProcessGroup returns a SysProcAttr structure configured to start a process with a new process group
func SysProcAttrForNewProcessGroup() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: 0x00000200, // CREATE_NEW_PROCESS_GROUP
	}
}
