package exeutils

import (
	"os"
	"strings"

	"github.com/thoas/go-funk"
)

const Extension = ".exe"

var exts = []string{".exe"}

func init() {
	PATHEXT := os.Getenv("PATHEXT")
	exts = funk.Uniq(funk.Map(strings.Split(PATHEXT, string(os.PathListSeparator)), strings.ToLower).([]string)).([]string)
}

