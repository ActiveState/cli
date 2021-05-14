// +build windows

package legacyupd

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func hideFile(path string) error {
	k32 := syscall.NewLazyDLL("kernel32.dll")
	setFileAttrs := k32.NewProc("SetFileAttributesW")

	uipPath := uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path)))
	r1, _, err := setFileAttrs.Call(uipPath, 2)
	if r1 == 0 && !errors.Is(err, windows.ERROR_SUCCESS) {
		return fmt.Errorf("Hide file (set attributes): %w", err)
	}

	return nil
}
