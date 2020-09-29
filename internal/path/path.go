package path

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ProvidesExecutable will search the given path for an executable.
// If filterByPath is not the empty string PathProvidesExec will only
// search paths with the filterByPath prefix
func ProvidesExecutable(filterByPath, exec, path string) bool {
	paths := splitPath(path)
	if filterByPath != "" {
		paths = filterPrefixed(filterByPath, paths)
	}
	paths = applySuffix(exec, paths)

	for _, p := range paths {
		if isExecutableFile(p) {
			return true
		}
	}
	return false
}

func splitPath(path string) []string {
	return strings.Split(path, string(os.PathListSeparator))
}

func filterPrefixed(prefix string, paths []string) []string {
	var ps []string
	for _, p := range paths {
		// Clean removes double slashes and relative path directories
		if strings.HasPrefix(filepath.Clean(p), filepath.Clean(prefix)) {
			ps = append(ps, p)
		}
	}
	return ps
}

func applySuffix(suffix string, paths []string) []string {
	for i, v := range paths {
		paths[i] = filepath.Join(v, suffix)
	}
	return paths
}

func isExecutableFile(name string) bool {
	f, err := os.Stat(name)
	if err != nil { // unlikely unless file does not exist
		return false
	}

	if runtime.GOOS == "windows" {
		return f.Mode()&0400 != 0
	}

	return f.Mode()&0110 != 0
}
