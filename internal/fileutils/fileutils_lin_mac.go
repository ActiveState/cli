//go:build !windows
// +build !windows

package fileutils

import (
	"os"
	"path/filepath"
	"strings"

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
	for !TargetExists(path) && path != "" {
		path = filepath.Dir(path)
	}
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

// SmartLink creates a link from src to target. On Linux and Mac this is just a symbolic link.
func SmartLink(src, dest string) error {
	var err error
	src, err = ResolvePath(src)
	if err != nil {
		return errs.Wrap(err, "Could not resolve src path")
	}
	dest, err = ResolvePath(dest)
	if err != nil {
		return errs.Wrap(err, "Could not resolve destination path")
	}
	return SymLink(src, dest)
}

// SmartUnlinkContents will unlink the contents of src to dest if the links exist
func SmartUnlinkContents(src, dest string) error {
	if !DirExists(dest) {
		return errs.New("dest dir does not exist: %s", dest)
	}

	var err error
	src, err = ResolvePath(src)
	if err != nil {
		return errs.Wrap(err, "Could not resolve src path")
	}
	dest, err = ResolvePath(dest)
	if err != nil {
		return errs.Wrap(err, "Could not resolve destination path")
	}

	entries, err := os.ReadDir(dest)
	if err != nil {
		return errs.Wrap(err, "Reading dir %s failed", dest)
	}
	for _, entry := range entries {
		realPath, err := filepath.EvalSymlinks(filepath.Join(dest, entry.Name()))
		if err != nil {
			return errs.Wrap(err, "Could not evaluate symlink of %s", entry.Name())
		}

		// Ensure we only delete this file if we can ensure that it comes from our src
		if !strings.HasPrefix(realPath, src) {
			return errs.New("File %s has unexpected link: %s", entry.Name(), realPath)
		}

		// Delete the link
		// No need to recurse here as we're dealing with symlinks
		if err := os.Remove(filepath.Join(dest, entry.Name())); err != nil {
			return errs.Wrap(err, "Could not unlink %s", entry.Name())
		}
	}

	return nil
}
