// +build !windows

package cmd

import "errors"

func notExistError() error {
	return errors.New("NOT_EXIST_ERROR")
}

func IsNotExistError(err error) bool {
	return err.Error() == "NOT_EXIST_ERROR"
}

func OpenKey(path string) (RegistryKey, error) {
	panic("Not supported outside of Windows, this only exists to facilitate unit tests")
	return nil, nil
}
