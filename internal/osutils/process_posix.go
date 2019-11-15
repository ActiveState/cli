// +build linux darwin

package osutils

import "syscall"


func SysProcAttrForNewProcessGroup() *syscall.SysProcAttr {
	return &syscall.SysProcAttr {
		SetSid: true,
	}
}
