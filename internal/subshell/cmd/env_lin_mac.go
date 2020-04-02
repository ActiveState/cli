// +build !windows

package cmd

func OpenKey(path string) (RegistryKey, error) {
	panic("Not supported outside of Windows, this only exists to facilitate unit tests")
	return nil, nil
}

func IsNotExistError(err error) bool {
	panic("Not supported outside of Windows, this only exists to facilitate unit tests")
	return false
}
