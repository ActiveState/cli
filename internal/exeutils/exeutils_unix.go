// +build !windows

package exeutils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
)

const Extension = ""

// PathForExecutable returns the first path from the PATH env var for which the executable exists
func PathForExecutable(executable string) string {
	candidates := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	for _, p := range candidates {
		if fileutils.TargetExists(filepath.Join(p, executable)) {
			return p
		}
	}
	return ""
}
