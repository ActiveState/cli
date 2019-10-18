package condition

import (
	"os"
	"strings"
)

var inTest = strings.HasSuffix(strings.TrimSuffix(os.Args[0], ".exe"), ".test") ||
	strings.Contains(os.Args[0], "/_test/")

// InTest returns true when the app is being tested
func InTest() bool {
	return inTest
}
