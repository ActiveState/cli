// +build windows

package fileutils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/gobuffalo/packr"
	"github.com/google/uuid"
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
		if strings.ToLower(ext) == strings.ToLower(pe) {
			return true
		}
	}
	return false
}

// IsWritable returns true if the given path is writable
func IsWritable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		logging.Error("Could not stat path: %s, got error: %v", path, err)
		return false
	}

	// Check if read-only bit is set
	if info.Mode().Perm()&(0222) == 0 {
		return false
	}

	box := packr.NewBox("../../assets/scripts")
	contents := box.String("IsWritable.ps1")
	scriptFile, err := WriteTempFile(
		"", "IsWritable*.ps1", []byte(contents), 0700,
	)
	if err != nil {
		logging.Error("Could not create temporary powershell file: %v", err)
		return false
	}

	cmd := exec.Command("powershell.exe", "-c", scriptFile, path)
	bytes, err := cmd.Output()
	if err != nil {
		logging.Debug("Could not determine if path: %s is writable, got error: %v", path, err)
		// Fallback on writing a tempfile
		return isWritableTempFile(path)
	}

	output := strings.TrimSpace(string(bytes))
	if output != "True" {
		logging.Debug("Path %s is not writable, got output: %s", path, output)
		return false
	}

	return true
}

func isWritableTempFile(path string) bool {
	fpath := filepath.Join(path, uuid.New().String())
	if err := Touch(fpath); err != nil {
		return false
	}

	if errr := os.Remove(fpath); errr != nil {
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
		if !errors.Is(err, os.ErrNotExist) {
			logging.Error("could not resolve long version of %s: %v", evalPath, err)
		}
		return filepath.Clean(evalPath), nil
	}

	return filepath.Clean(longPath), nil
}

func HideFile(path string) error {
	k32 := syscall.NewLazyDLL("kernel32.dll")
	setFileAttrs := k32.NewProc("SetFileAttributesW")

	uipPath := uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path)))
	r1, _, err := setFileAttrs.Call(uipPath, 2)
	if r1 == 0 && !errors.Is(err, windows.ERROR_SUCCESS) {
		return fmt.Errorf("Hide file (set attributes): %w", err)
	}

	return nil
}
