//go:build !windows
// +build !windows

package fileutils

// GetShortPathName returns the Windows short path
// This function does not alter the path in any way on Linux or MacOS
func GetShortPathName(path string) (string, error) {
	return path, nil
}

// GetLongPathName returns the Windows long path (ie. ~1 notation is expanded)
// This function does not alter the path in any way on Linux or MacOS
func GetLongPathName(path string) (string, error) {
	return path, nil
}
