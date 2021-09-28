package exeutils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/thoas/go-funk"
)

const Extension = ".exe"

// FindExecutableOnOSPath returns the first path from the PATH env var for which the executable exists
func FindExecutableOnOSPath(executable string) string {
	return FindExecutableOnPath(executable, os.Getenv("PATH"))
}

func FindExecutableOnPath(executable, PATH string) string {
	return findExecutable(executable, PATH, os.Getenv("PATHEXT"))
}

func findExecutable(executable, PATH, PATHEXT string) string {
	candidates := strings.Split(PATH, string(os.PathListSeparator))
	var exts []string // list of extensions to look for
	pathexts := funk.Map(strings.Split(PATHEXT, string(os.PathListSeparator)), strings.ToLower).([]string)
	// if executable has valid extension for an executable file, we have to check for its existence without appending more extensions
	if funk.ContainsString(pathexts, strings.ToLower(filepath.Ext(executable))) {
		exts = append(exts, "")
	}
	exts = append(exts, pathexts...)
	for _, p := range candidates {
		for _, ext := range exts {
			fp := filepath.Clean(filepath.Join(p, executable+ext))
			if fileutils.TargetExists(fp) {
				return fp
			}
		}
	}
	return ""
}
