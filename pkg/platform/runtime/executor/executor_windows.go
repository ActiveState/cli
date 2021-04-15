package executor

import (
	"os"
	"strings"
)

func NameForExe(exe string) string {
	exts := strings.Split(strings.ToLower(os.Getenv("PATHEXT")), ";")
	lowerExe := strings.ToLower(exe)
	for _, ext := range exts {
		if strings.HasSuffix(lowerExe, strings.ToLower(ext)) {
			return exe[0:len(exe)-len(ext)] + ".bat"
		}
	}
	return exe + ".bat"
}
