package condition

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/thoas/go-funk"
)

var inTest = strings.HasSuffix(strings.TrimSuffix(os.Args[0], ".exe"), ".test") ||
	strings.Contains(os.Args[0], "/_test/") || funk.Contains(os.Args, "-test.v")

// InUnitTest returns true when the app is being tested
func InUnitTest() bool {
	return inTest
}

func InTest() bool {
	return InUnitTest() || os.Getenv(constants.E2ETestEnvVarName) == "true"
}

func OnCI() bool {
	return os.Getenv("CI") != "" || os.Getenv("BUILDER_OUTPUT") != ""
}

func BuiltViaCI() bool {
	return constants.OnCI == "true"
}

func InStateSvc() bool {
	return strings.HasSuffix(os.Args[0], constants.StateSvcCmd)
}
