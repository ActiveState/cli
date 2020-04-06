// +build !windows

package cmd

import "errors"

type errNotExist struct{}

func (e errNotExist) Error() string {
	return "ErrNotExist"
}

var ErrNotExist errNotExist

func IsNotExistError(err error) bool {
	return errors.Is(err, ErrNotExist)
}

func OpenKey(path string) (RegistryKey, error) {
	panic("Not supported outside of Windows, this only exists to facilitate unit tests")
	return nil, nil
}
