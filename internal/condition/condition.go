package condition

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/thoas/go-funk"
)

type Configurable interface {
	GetBool(s string) bool
}

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

func IsLTS() bool {
	return strings.HasPrefix(constants.BranchName, "LTS")
}

func BuiltViaCI() bool {
	return constants.OnCI == "true"
}

func BuiltOnDevMachine() bool {
	return !BuiltViaCI()
}

func InActiveStateCI() bool {
	return os.Getenv(constants.ActiveStateCIEnvVarName) == "true"
}

func OptInUnstable(cfg Configurable) bool {
	if v := os.Getenv(constants.OptinUnstableEnvVarName); v != "" {
		return v == "true"
	}
	return cfg.GetBool(constants.UnstableConfig)
}

func IsNetworkingError(err error) bool {
	msg := errs.JoinMessage(err)
	switch {
	case strings.Contains(msg, "no such host"):
		return true
	case strings.Contains(msg, "no route to host"):
		return true
	}
	return false
}
