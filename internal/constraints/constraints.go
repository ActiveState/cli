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

// Returns whether or not the given platform is constrained by the given
// constraint name.
// If the constraint name is prefixed by "-", returns the converse.
func platformIsConstrainedByConstraintName(platform projectfile.Platform, name string) bool {
	if platform.Name == strings.TrimLeft(name, "-") {
		if match(platform.Os, osName()) &&
			match(platform.Version, osVersion()) &&
			match(platform.Architecture, osArchitecture()) &&
			match(platform.Libc, osLibc()) &&
			match(platform.Compiler, osCompiler()) {
			if strings.HasPrefix(name, "-") {
				return true
			}
		} else if !strings.HasPrefix(name, "-") {
			return true
		}
	}
	return false
}

// Returns whether or not the current platform is constrained by the given
// named constraints, which are defined in the given project configuration.
func platformIsConstrained(constraintNames string, project *projectfile.Project) bool {
	for _, name := range strings.Split(constraintNames, ",") {
		for _, platform := range project.Platforms {
			if platformIsConstrainedByConstraintName(platform, name) {
				return true
			}
		}
	}
	return false
}

// Returns whether or not the current environment is constrained by the given
// constraints.
func environmentIsConstrained(constraints string) bool {
	constraintList := strings.Split(constraints, ",")
	for _, constraint := range constraintList {
		if constraint == os.Getenv(constants.EnvironmentEnvVarName) {
			return false
		}
	}
	return true
}

// IsConstrained returns whether or not the given constraints are constraining
// based on given project configuration.
func IsConstrained(constraint projectfile.Constraint, project *projectfile.Project) bool {
	return platformIsConstrained(constraint.Platform, project) ||
		environmentIsConstrained(constraint.Environment)
}
