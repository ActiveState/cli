package exeutils

import (
	"os"
)

const Extension = ".exe"

var exts = []string{".exe"}

func init() {
	PATHEXT := os.Getenv("PATHEXT")
	exts = []string{} /*funk.Uniq(funk.Map(strings.Split(PATHEXT, string(os.PathListSeparator)), strings.ToLower).([]string)).([]string)*/
}
