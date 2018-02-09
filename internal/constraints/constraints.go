package constraints

import (
	"os"
	"runtime"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
)

// Mainly for testing.
var osNameOverride, osVersionOverride, osArchitectureOverride, osLibcOverride, osCompilerOverride string

// Returns whether or not the expected string matches the actual string,
// considering an empty expectation to be a match.
func match(expected string, actual string) bool {
	return expected == "" || expected == actual
}

func osName() string {
	if osNameOverride != "" {
		return osNameOverride
	}
	return runtime.GOOS
}

func osVersion() string {
	if osVersionOverride != "" {
		return osVersionOverride
	}
	return "" // TODO
}

func osArchitecture() string {
	if osArchitectureOverride != "" {
		return osArchitectureOverride
	}
	return runtime.GOARCH
}

func osLibc() string {
	if osLibcOverride != "" {
		return osLibcOverride
	}
	return "" // TODO
}

func osCompiler() string {
	if osCompilerOverride != "" {
		return osCompilerOverride
	}
	return "" // TODO
}

// Returns whether or not the given constraint matches the given project
// configuration.
func matchesPlatform(constraints string, project *projectfile.Project) bool {
	constraintList := strings.Split(constraints, ",")
	for _, constraint := range constraintList {
		for _, platform := range project.Platforms {
			if platform.Name == strings.TrimLeft(constraint, "-") {
				if match(platform.Os, osName()) &&
					match(platform.Version, osVersion()) &&
					match(platform.Architecture, osArchitecture()) &&
					match(platform.Libc, osLibc()) &&
					match(platform.Compiler, osCompiler()) {
					if strings.HasPrefix(constraint, "-") {
						return false
					}
				} else if !strings.HasPrefix(constraint, "-") {
					return false
				}
			}
		}
	}
	return true
}

// Returns whether or not the given constraint matches the given project
// configuration.
func matchesEnvironment(constraints string) bool {
	constraintList := strings.Split(constraints, ",")
	for _, constraint := range constraintList {
		if constraint == os.Getenv(constants.EnvironmentEnvVarName) {
			return true
		}
	}
	return false
}

// MatchesConstraints returns whether or not the given constraints match the
// given project configuration.
func MatchesConstraints(constraint projectfile.Constraint, project *projectfile.Project) bool {
	return matchesPlatform(constraint.Platform, project) &&
		matchesEnvironment(constraint.Environment)
}
