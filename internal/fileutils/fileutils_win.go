// +build windows

package fileutils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/thoas/go-funk"
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
	cmd := exec.Command("powershell", "-c", fmt.Sprintf("(Get-Acl %s).AccessToString | findstr \"$env:USERNAME\"", path))

	out, err := cmd.Output()
	if err != nil {
		logging.Debug(fmt.Sprintf("Path %s is not writable, got error: %v", path, err))
		return false
	}

	if funk.Contains(string(out), "FullControl") {
		// TODO: Add more checks for values from here:
		// https://docs.microsoft.com/en-us/dotnet/api/system.security.accesscontrol.filesystemrights?view=dotnet-plat-ext-3.1
		return true
	}

	return false
}
