// +build !windows

package osutils

func IsWindowsAdmin() (bool, error) {
	return false, nil
}