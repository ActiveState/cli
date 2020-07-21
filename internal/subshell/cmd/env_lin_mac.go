// +build !windows

package cmd

import "errors"

func notExistError() error {
	return errors.New("NOT_EXIST_ERROR")
}

func IsNotExistError(err error) bool {
	return err.Error() == "NOT_EXIST_ERROR"
}

func OpenUserKey(path string) (RegistryKey, error) {
	panic("Not supported outside of Windows, this only exists to facilitate unit tests")
	return nil, nil
}

func OpenSystemKey(path string) (RegistryKey, error) {
	panic("Not supported outside of Windows, this only exists to facilitate unit tests")
	return nil, nil
}

func (c *CmdEnv) propagate() {
}

func setStringValue(key RegistryKey, name string, valType uint32, value string) error {
	panic("Not supported outside of Windows, this only exists to facilitate unit tests")
	return nil
}
