package osutils

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

func PropagateEnv() {
	// Note: Always use SendNotifyMessageW here, as SendMessageW can hang forever (https://stackoverflow.com/a/1956702)
	syscall.NewLazyDLL("user32.dll").NewProc("SendNotifyMessageW").Call(HwndBroadcast, WmSettingChange, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("ENVIRONMENT"))))
}

func SetStringValue(key RegistryKey, name string, valType uint32, value string) error {
	f := key.SetStringValue
	if valType == registry.EXPAND_SZ {
		f = key.SetExpandStringValue
	}
	return f(name, value)
}
