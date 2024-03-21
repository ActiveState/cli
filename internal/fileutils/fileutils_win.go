//go:build windows
// +build windows

package fileutils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"golang.org/x/sys/windows"
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
	for _, pe := range pathExts {
		// pathext entries have `.` and are capitalize
		if strings.EqualFold(ext, pe) {
			return true
		}
	}
	return false
}

// IsWritable returns true if the given path is writable
func IsWritable(path string) bool {
	for !TargetExists(path) && path != "" {
		path = filepath.Dir(path)
	}

	info, err := os.Stat(path)
	if err != nil {
		logging.Debug("os.Stat %s failed", path)
		return false
	}

	// Check if the user bit is enabled in file permission
	if info.Mode().Perm()&(1<<(uint(7))) == 0 {
		logging.Debug("Write permission bit is not set on this file for user")
		return false
	}

	return true
}

// ResolveUniquePath gets the absolute location of the provided path
// with the best effort attempt to produce the same result for all possible paths to the
// given target.
func ResolveUniquePath(path string) (string, error) {
	evalPath, err := ResolvePath(filepath.Clean(path))
	if err != nil {
		return "", errs.Wrap(err, "cannot resolve path")
	}

	longPath, err := GetLongPathName(evalPath)
	if err != nil {
		// GetLongPathName can fail on unsupported file-systems or if evalPath is not a physical path.
		// => just log the error (unless err due to file not existing) and resume with resolved path
		if !errors.Is(err, os.ErrNotExist) && !errs.Matches(err, os.ErrNotExist) {
			multilog.Error("could not resolve long version of %s: %v", evalPath, err)
		}
		return filepath.Clean(evalPath), nil
	}

	return filepath.Clean(longPath), nil
}

func HideFile(path string) error {
	k32 := syscall.NewLazyDLL("kernel32.dll")
	setFileAttrs := k32.NewProc("SetFileAttributesW")

	utfPath, err := syscall.UTF16FromString(path)
	if err != nil {
		return fmt.Errorf("Hide file (UTF16 conversion): %w", err)
	}
	uipPath := uintptr(unsafe.Pointer(&utfPath[0]))
	r1, _, err := setFileAttrs.Call(uipPath, 2)
	if r1 == 0 && !errors.Is(err, windows.ERROR_SUCCESS) {
		return fmt.Errorf("Hide file (set attributes): %w", err)
	}

	return nil
}
