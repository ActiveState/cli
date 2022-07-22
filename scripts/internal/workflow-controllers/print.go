package wc

import (
	"fmt"
	"strings"
)

var printDepth = 0

func sprint(depth int, msg string, args ...interface{}) string {
	msg = fmt.Sprintf(msg, args...)
	prefix := ""
	if depth > 0 {
		prefix = "|- "
	}
	indent := strings.Repeat("  ", depth)
	msg = strings.Replace(msg, "\n", "\n   "+indent, -1)
	return fmt.Sprintf(indent + prefix + msg)
}

func Print(msg string, args ...interface{}) {
	fmt.Println(sprint(printDepth, msg, args...))
}

func PrintStart(description string, args ...interface{}) func() {
	Print(description+"..", args...)
	printDepth++
	return func() {
		printDepth--
		Print("Done\n")
	}
}
