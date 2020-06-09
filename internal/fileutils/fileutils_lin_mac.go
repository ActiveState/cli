// +build !windows

package fileutils

import (
	"os"
	"path/filepath"

	"github.com/google/uuid"
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
	fpath := filepath.Join(path, uuid.New().String())
	if fail := Touch(fpath); fail != nil {
		return false
	}

	if errr := os.Remove(fpath); errr != nil {
		return false
	}

	return true
}
