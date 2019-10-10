package condition

import (
	"flag"
	"os"
	"strings"
)

var inTest = strings.HasSuffix(os.Args[0], ".test") ||
	strings.Contains(os.Args[0], "/_test/") ||
	flag.Lookup("test.v") != nil

// InTest returns true when the app is being tested
func InTest() bool {
	if inTest {
		return inTest
	}

	for _, flag := range os.Args {
		if strings.Contains(flag, "test.timeout") {
			return true
		}
		if strings.Contains(flag, "test.logfile") {
			return true
		}
	}

	return false
}
