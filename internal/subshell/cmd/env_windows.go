package cmd

import (
	"errors"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	HwndBroadcast   = uintptr(0xffff)
	WmSettingChange = uintptr(0x001A)
)

func notExistError() error {
	return registry.ErrNotExist
}

func OpenUserKey(path string) (RegistryKey, error) {
	return registry.OpenKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
}

func OpenSystemKey(path string) (RegistryKey, error) {
	return registry.OpenKey(registry.LOCAL_MACHINE, path, registry.ALL_ACCESS)
}

func IsNotExistError(err error) bool {
	return errors.Is(err, registry.ErrNotExist)
}

func (c *CmdEnv) propagate() {
	syscall.NewLazyDLL("user32.dll").NewProc("SendMessageW").Call(HwndBroadcast, WmSettingChange, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("ENVIRONMENT"))))
}
