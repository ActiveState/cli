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
		if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, os.ErrNotExist) {
			multilog.Error("could not resolve long version of %s: %v", evalPath, err)
		}
		return filepath.Clean(evalPath), nil
	}

	return filepath.Clean(longPath), nil
}

func HideFile(path string) error {
	k32 := syscall.NewLazyDLL("kernel32.dll")
	setFileAttrs := k32.NewProc("SetFileAttributesW")

	utfPath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("Hide file (UTF16 conversion): %w", err)
	}
	uipPath := uintptr(unsafe.Pointer(utfPath))
	r1, _, err := setFileAttrs.Call(uipPath, 2)
	if r1 == 0 && !errors.Is(err, windows.ERROR_SUCCESS) {
		return fmt.Errorf("Hide file (set attributes): %w", err)
	}

	return nil
}

// SmartLink creates a link from src to target. MS decided to support Symlinks but only if you opt into developer mode (go figure),
// which we cannot reasonably force on our users. So on Windows we will instead create dirs and hardlinks.
func SmartLink(src, dest string) error {
	if TargetExists(dest) {
		return errs.New("target already exists: %s", dest)
	}

	if DirExists(src) {
		if err := os.MkdirAll(dest, 0755); err != nil {
			return errs.Wrap(err, "could not create directory %s", dest)
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return errs.Wrap(err, "could not read directory %s", src)
		}
		for _, entry := range entries {
			if err := SmartLink(filepath.Join(src, entry.Name()), filepath.Join(dest, entry.Name())); err != nil {
				return errs.Wrap(err, "sub link failed")
			}
		}
		return nil
	}

	if err := os.Link(src, dest); err != nil {
		return errs.Wrap(err, "could not link %s to %s", src, dest)
	}
	return nil
}

// SmartUnlinkContents will unlink the contents of src to dest if the links exist
// WARNING: on windows smartlinks are hard links, and relating hard links back to their source is non-trivial, so instead
// we just delete the target path. If the user modified the target in any way their changes will be lost.
func SmartUnlinkContents(src, dest string) error {
	if !DirExists(dest) {
		return errs.New("dest dir does not exist: %s", dest)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return errs.Wrap(err, "Reading dir %s failed", dest)
	}
	for _, entry := range entries {
		path := filepath.Join(dest, entry.Name())
		if !TargetExists(path) {
			continue
		}

		if err := os.RemoveAll(path); err != nil {
			return errs.Wrap(err, "Could not delete %s", path)
		}
	}

	return nil
}
