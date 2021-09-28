package exeutils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/thoas/go-funk"
)

const Extension = ".exe"

// PathForExecutable returns the first path from the PATH env var for which the executable exists
func PathForExecutable(executable string) string {
	candidates := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	var exts []string // list of extensions to look for
	pathexts := strings.Split(os.Getenv("PATHEXT"), string(os.PathListSeparator))
	// if executable has valid extension for an executable file, we have to check for its existence without appending more extensions
	if funk.ContainsString(pathexts, filepath.Ext(executable)) {
		exts = append(exts, "")
	}
	exts = append(exts, pathexts...)
	for _, p := range candidates {
		for _, ext := range exts {
			if fileutils.TargetExists(filepath.Join(p, executable+ext)) {
				return p
			}
		}
	}
	return ""
}
