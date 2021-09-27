package exeutils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
)

const Extension = ".exe"

// PathForExecutable returns the first path from the PATH env var for which the executable exists
func PathForExecutable(executable string) string {
	candidates := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	exts := append([]string{""}, strings.Split(os.Getenv("PATHEXT"), string(os.PathListSeparator))...)
	for _, p := range candidates {
		for _, ext := range exts {
			if fileutils.TargetExists(filepath.Join(p, executable+ext)) {
				return p
			}
		}
	}
	return ""
}
