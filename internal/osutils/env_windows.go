package osutils

import (
	"errors"
	"syscall"
	"unsafe"

	"github.com/ActiveState/cli/internal/errs"
	"golang.org/x/sys/windows/registry"
)

const (
	HwndBroadcast   = uintptr(0xffff)
	WmSettingChange = uintptr(0x001A)
)

func NotExistError() error {
	return registry.ErrNotExist
}

func OpenUserKey(path string) (RegistryKey, error) {
	return registry.OpenKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
}

func OpenSystemKey(path string) (RegistryKey, error) {
	return registry.OpenKey(registry.LOCAL_MACHINE, path, registry.ALL_ACCESS)
}

func CreateUserKey(path string) (RegistryKey, bool, error) {
	return registry.CreateKey(registry.USERS, path, registry.ALL_ACCESS)
}

func CreateCurrentUserKey(path string) (RegistryKey, bool, error) {
	return registry.CreateKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
}

func IsNotExistError(err error) bool {
	return errors.Is(err, registry.ErrNotExist)
}

func PropagateEnv() error {
	lparam, err := syscall.UTF16PtrFromString("ENVIRONMENT")
	if err != nil {
		return errs.Wrap(err, "Could convert UTF16 pointer from string")
	}
	// Note: Always use SendNotifyMessageW here, as SendMessageW can hang forever (https://stackoverflow.com/a/1956702)
	result, _, err := syscall.NewLazyDLL("user32.dll").NewProc("SendNotifyMessageW").Call(HwndBroadcast, WmSettingChange, 0, uintptr(unsafe.Pointer(lparam)))

	if result == 0 {
		return errs.Wrap(err, "sendNotifyMessageW failed.")
	}
	return nil
}

func SetStringValue(key RegistryKey, name string, valType uint32, value string) error {
	f := key.SetStringValue
	if valType == registry.EXPAND_SZ {
		f = key.SetExpandStringValue
	}
	return f(name, value)
}
