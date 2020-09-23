// +build !windows

package fileutils

import (
	"os"

	"golang.org/x/sys/unix"
)

const LineEnd = "\n"

// IsExecutable determines if the file at the given path has any execute permissions.
// This function does not care whether the current user can has enough privilege to
// execute the file.
func IsExecutable(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && (stat.Mode()&(0111) > 0)
}

// IsWritable returns true if the given path is writable
func IsWritable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}
