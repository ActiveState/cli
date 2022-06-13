package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	exec := os.Args[0]
	exec = `"` + exec + `"`
	fmt.Println("Exec:", exec)
	clean := trimQuotes(exec)
	fmt.Println("Clean:", clean)
	another := strings.Trim(exec, "\"")
	fmt.Println("Another:", another)
}

// Remove quotes of the source string
func trimQuotes(s string) string {
	if len(s) >= 2 {
		if s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
	}
	return s
}
