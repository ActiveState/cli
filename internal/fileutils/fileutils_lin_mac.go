// +build !windows

package fileutils

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
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

// ResolveUniquePath gets the absolute location of the provided path
// with the best effort attempt to produce the same result for all possible paths to the
// given target.
func ResolveUniquePath(path string) (string, error) {
	// "un-clean" file paths seem to confuse the EvalSymlinks function on MacOS, so we have to clean the path here.
	evalPath, err := ResolvePath(filepath.Clean(path))
	if err != nil {
		return "", errs.Wrap(err, "cannot resolve path")
	}

	return filepath.Clean(evalPath), nil
}

func HideFile(path string) error {
	return nil
}
