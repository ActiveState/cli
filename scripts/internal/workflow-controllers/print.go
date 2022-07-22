package wc

import (
	"fmt"
	"strings"
)

var printDepth = 0

func Print(msg string, args ...interface{}) {
	prefix := ""
	if printDepth > 0 {
		prefix = "|- "
	}
	indent := strings.Repeat("  ", printDepth)
	msg = strings.Replace(msg, "\n", indent+"\n", -1)
	fmt.Printf(indent + prefix + fmt.Sprintf(msg+"\n", args...))
}

func PrintStart(description string, args ...interface{}) func() {
	Print(description+"..", args...)
	printDepth++
	return func() {
		printDepth--
		Print("Done\n")
	}
}
