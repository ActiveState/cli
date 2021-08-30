package condition

import (
	"os"
	"strings"

	"github.com/thoas/go-funk"
)

var inTest = strings.HasSuffix(strings.TrimSuffix(os.Args[0], ".exe"), ".test") ||
	strings.Contains(os.Args[0], "/_test/") || funk.Contains(os.Args, "-test.v")

// InTest returns true when the app is being tested
func InTest() bool {
	return inTest
}

func OnCI() bool {
	return os.Getenv("CI") != "" || os.Getenv("BUILDER_OUTPUT") != ""
}