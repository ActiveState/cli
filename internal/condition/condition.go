package condition

import (
	"flag"
	"os"
	"strings"
)

var inTest = strings.HasSuffix(os.Args[0], ".test") ||
	strings.Contains(os.Args[0], "/_test/") ||
	flag.Lookup("test.v") != nil

func init() {
	if inTest {
		return
	}

	for _, flag := range os.Args {
		if strings.Contains(flag, "test.timeout") {
			inTest = true
		}
		if strings.Contains(flag, "test.logfile") {
			inTest = true
		}
	}
}

// InTest returns true when the app is being tested
func InTest() bool {
	return inTest
}
