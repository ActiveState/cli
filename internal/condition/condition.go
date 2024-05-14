package condition

import (
	"errors"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/thoas/go-funk"
)

type Configurable interface {
	GetBool(s string) bool
}

var osArgStr = strings.Join(os.Args, " ")

var isTestInvocation = strings.HasSuffix(strings.TrimSuffix(os.Args[0], ".exe"), ".test") ||
	strings.Contains(os.Args[0], "/_test/") || funk.Contains(os.Args, "-test.v")

var inUnitTest = !strings.Contains(osArgStr, "IntegrationTestSuite") &&
	!strings.Contains(osArgStr, "integration.test") &&
	isTestInvocation

// InUnitTest returns true when the app is being tested
func InUnitTest() bool {
	return inUnitTest
}

func InTest() bool {
	return InUnitTest() || os.Getenv(constants.E2ETestEnvVarName) == "true"
}

func OnCI() bool {
	return os.Getenv("CI") != "" || os.Getenv("BUILDER_OUTPUT") != ""
}

func IsLTS() bool {
	return strings.HasPrefix(constants.ChannelName, "LTS")
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
	switch {
	case strings.Contains(err.Error(), "no such host"):
		return true
	case strings.Contains(err.Error(), "no route to host"):
		return true
	}
	if subErr := errors.Unwrap(err); subErr != nil {
		return IsNetworkingError(subErr)
	}
	unwrapped, ok := err.(interface{ Unwrap() []error })
	if ok {
		subErrs := unwrapped.Unwrap()
		if len(subErrs) > 0 {
			for _, subErr := range subErrs {
				if IsNetworkingError(subErr) {
					return true
				}
			}
		}
	}
	return false
}
