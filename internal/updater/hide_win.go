// +build windows

package updater

import (
	"fmt"
	"syscall"
	"unsafe"
)

func hideFile(path string) error {
	k32 := syscall.NewLazyDLL("kernel32.dll")
	setFileAttrs := k32.NewProc("SetFileAttributesW")

	uipPath := uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path)))
	r1, _, err := setFileAttrs.Call(uipPath, 2)
	if r1 == 0 && err != 0 {
		return fmt.Errorf("Hide file (set attributes): %w", err)
	}

	return nil
}
