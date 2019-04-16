// +build windows

package fileutils

import (
	"os"
	"strings"
)

// IsExecutable determines if the file at the given path has any execute permissions.
// This function does not care whether the current user can has enough privilege to
// execute the file.
func IsExecutable(path string) bool {
	pathSplit := strings.Split(path, ".")
	exe := pathSplit[len(pathSplit)-1]
	if exe == "exe" {
		return true
	}

	pathExts := strings.Split(os.Getenv("PATHEXT"), ";")
	for _, ext := range pathExts {
		// pathext entries have `.` and are capitalize
		if strings.ToLower(exe) == strings.ToLower(ext)[1:] {
			return true
		}
	}
	return false
}
