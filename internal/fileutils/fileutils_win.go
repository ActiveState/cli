// +build windows

package fileutils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/gobuffalo/packr"
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
	box := packr.NewBox("../../assets/scripts")
	contents := box.String("IsWritable.ps1")
	scriptFile, fail := WriteTempFile(
		"", fmt.Sprintf("%s*%s", "IsWritable", ".ps1"), []byte(contents), 0700,
	)
	if fail != nil {
		return false
	}

	cmd := exec.Command("powershell.exe", "-c", scriptFile, path)
	err := cmd.Run()
	if err != nil {
		logging.Debug("Path %s is not writable, got error %v", path, err)
		return false
	}

	return true
}
