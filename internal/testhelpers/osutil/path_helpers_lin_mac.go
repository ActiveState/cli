// +build !windows

package osutil

// GetLongPath name returns the Windows long path (ie. ~1 notation is expanded)
// This function does not alter the path in any way on Linux or MacOS
func GetLongPathName(path string) (string error) {
	return path, nil
}
