package condition

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/thoas/go-funk"
)

var inTest = strings.HasSuffix(strings.TrimSuffix(os.Args[0], ".exe"), ".test") ||
	strings.Contains(os.Args[0], "/_test/") || funk.Contains(os.Args, "-test.v")

// InUnitTest returns true when the app is being tested
func InUnitTest() bool {
	return inTest
}

func OnCI() bool {
	return os.Getenv("CI") != "" || os.Getenv("BUILDER_OUTPUT") != ""
}

func BuiltViaCI() bool {
	return rtutils.BuiltViaCI
}