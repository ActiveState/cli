// +build !windows

package osutils

import "errors"

func NotExistError() error {
	return errors.New("NOT_EXIST_ERROR")
}

func IsNotExistError(err error) bool {
	return err.Error() == "NOT_EXIST_ERROR"
}

func OpenUserKey(path string) (RegistryKey, error) {
	panic("Not supported outside of Windows, this only exists to facilitate unit tests")
}

func OpenSystemKey(path string) (RegistryKey, error) {
	panic("Not supported outside of Windows, this only exists to facilitate unit tests")
}

func CreateUserKey(path string) (RegistryKey, bool, error) {
	panic("Not supported outside of Windows, this only exists to facilitate unit tests")
}

func CreateCurrentUserKey(path string) (RegistryKey, bool, error) {
	panic("Not supported outside of Windows, this only exists to facilitate unit tests")
}

func PropagateEnv() error {
	return nil
}

func SetStringValue(key RegistryKey, name string, valType uint32, value string) error {
	return key.SetStringValue(name, value)
}
