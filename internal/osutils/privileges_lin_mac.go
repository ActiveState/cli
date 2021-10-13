// +build !windows

package osutils

func IsAdmin() (bool, error) {
	return false, nil // Currently we only care about this on Windows
}