// +build !windows

package exeutils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
)

const Extension = ""

// FindExecutableOnOSPath returns the first path from the PATH env var for which the executable exists
func FindExecutableOnOSPath(executable string) string {
	return FindExecutableOnPath(executable, os.Getenv("PATH"))
}

func FindExecutableOnPath(executable string, PATH string) string {
	return findExecutables(executable, PATH, fileutils.TargetExists)
}

func findExecutables(executable, PATH string, fileExists func(string) bool) string {
	candidates := strings.Split(PATH, string(os.PathListSeparator))
	for _, p := range candidates {
		fp := filepath.Clean(filepath.Join(p, executable))
		if fileExists(fp) {
			return fp
		}
	}
	return ""
}
