// +build windows

package osutils

import "syscall"

func SysProcAttrForNewProcessGroup() *syscall.SysProcAttr {
	return &syscall.SysProcAttr {
		CreationFlags: 0x00000200, // CREATE_NEW_PROCESS_GROUP
	}
}
