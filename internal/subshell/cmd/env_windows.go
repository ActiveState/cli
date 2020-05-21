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

func OpenKey(path string) (RegistryKey, error) {
	return registry.OpenKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
}

func IsNotExistError(err error) bool {
	return errors.Is(err, registry.ErrNotExist)
}

func (c *CmdEnv) propagate() {
	// Note: Always use SendNotifyMessageW here, as SendMessageW can hang forever (https://stackoverflow.com/a/1956702)
	syscall.NewLazyDLL("user32.dll").NewProc("SendNotifyMessageW").Call(HwndBroadcast, WmSettingChange, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("ENVIRONMENT"))))
}
