// +build windows

package deploy

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/logging"
)

func isWritable(path string) bool {
	// Avoid writing to paths that require elevated privledges
	avoidPaths := []string{
		"C:\\Windows",
		"C:\\Program Files",
		"C:\\Program Files (x86)",
	}

	info, err := os.Stat(path)
	if err != nil {
		logging.Error("Could not stat path: %s, got error: %v", path, err)
		return false
	}
	if !info.IsDir() {
		return false
	}

	// Check if the user bit is enabled in file permission
	if info.Mode().Perm()&(1<<(uint(7))) == 0 {
		logging.Debug("Write permission bit is not set on: %s", path)
		return false
	}

	for _, a := range avoidPaths {
		if strings.HasPrefix(path, a) {
			return false
		}
	}

	return true
}
