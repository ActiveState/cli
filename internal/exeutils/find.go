package exeutils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/thoas/go-funk"
)

// FindExeOnPATH returns the first path from the PATH env var for which the executable exists
func FindExeOnPATH(executable string) string {
	exes := findExes(executable, os.Getenv("PATH"), fileutils.TargetExists, nil)
	if len(exes) == 0 {
		return ""
	}
	return exes[0]
}

// FindExeOnPATH returns the first path from the PATH env var for which the executable exists
func FilterExesOnPATH(executable string, PATH string, filter func(exe string) bool) []string {
	return findExes(executable, PATH, fileutils.TargetExists, filter)
}

func FindExeInside(executable string, PATH string) string {
	exes := findExes(executable, PATH, fileutils.TargetExists, nil)
	if len(exes) == 0 {
		return ""
	}
	return exes[0]
}

func findExes(executable string, PATH string, fileExists func(string) bool, filter func(exe string) bool) []string {
	var exts = exts
	// if executable has valid extension for an executable file, we have to check for its existence without appending more extensions
	if funk.ContainsString(exts, strings.ToLower(filepath.Ext(executable))) {
		exts = []string{""}
	}

	result := []string{}
	candidates := funk.Uniq(strings.Split(PATH, string(os.PathListSeparator))).([]string)
	for _, p := range candidates {
		for _, ext := range exts {
			fp := filepath.Clean(filepath.Join(p, executable+ext))
			if fileExists(fp) && (filter == nil || filter(fp)) {
				result = append(result, fp)
			}
		}
	}
	return result
}

func findExe(executable string, PATH string, fileExists func(string) bool, filter func(exe string) bool) string {
	r := findExes(executable, PATH, fileExists, filter)
	if len(r) > 0 {
		return r[0]
	}
	return ""
}
