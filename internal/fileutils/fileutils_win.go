// +build windows

package fileutils

import (
	"os"
	"path/filepath"
	"strings"
)

const LineEnd = "\r\n"

// IsExecutable determines if the file at the given path has any execute permissions.
// This function does not care whether the current user can has enough privilege to
// execute the file.
func IsExecutable(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".exe" {
		return true
	}

	pathExts := strings.Split(os.Getenv("PATHEXT"), ";")
	for _, ext := range pathExts {
		// pathext entries have `.` and are capitalize
		if strings.ToLower(ext) == strings.ToLower(ext)[1:] {
			return true
		}
	}
	return false
}
