// +build windows

package fileutils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/gobuffalo/packr"
	"github.com/google/uuid"
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
	info, err := os.Stat(os.Args[1])
	if err != nil {
		logging.Error("Could not stat path: %s, got error: %v", path, err)
		return false
	}

	// Check if read-only bit is set
	if info.Mode().Perm()&(0|1<<(uint(7))|1<<(uint(4))|1<<(uint(1))) == 0 {
		return false
	}

	box := packr.NewBox("../../assets/scripts")
	contents := box.String("IsWritable.ps1")
	scriptFile, fail := WriteTempFile(
		"", "IsWritable*.ps1", []byte(contents), 0700,
	)
	if fail != nil {
		logging.Error("Could not create temporary powershell file: %v", fail)
		return false
	}

	cmd := exec.Command("powershell.exe", "-c", scriptFile, path)
	err = cmd.Run()
	if err != nil {
		logging.Debug("Path %s is not writable, got error %v", path, err)
		return false
	}

	return true
}

func isWritableTempFile(path string) bool {
	fpath := filepath.Join(path, uuid.New().String())
	if fail := Touch(fpath); fail != nil {
		return false
	}

	if errr := os.Remove(fpath); errr != nil {
		return false
	}

	return true
}
