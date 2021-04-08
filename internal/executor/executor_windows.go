package executor

import "strings"

func nameExecutor(exe string) string {
	exts := strings.Split(strings.ToLower(os.Getenv("PATHEXT")), ";")
	lowerExe := strings.ToLower(exe)
	for _, ext := range exts {
		if strings.HasSuffix(lowerExe, strings.ToLower(ext)) {
			return exe[0 : len(exe)-ext]
		}
	}
	return exe + ".bat"
}
