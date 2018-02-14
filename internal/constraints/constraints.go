package constraints

import (
	"os"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"

	"github.com/ActiveState/go-sysinfo/sysinfo"
)

// Returns whether or not the expected string matches the actual string,
// considering an empty expectation to be a match.
func match(expected string, actual string) bool {
	return expected == "" || expected == actual
}

// Returns whether or not the given platform is constrained by the given
// constraint name.
// If the constraint name is prefixed by "-", returns the converse.
func platformIsConstrainedByConstraintName(platform projectfile.Platform, name string) bool {
	if platform.Name == strings.TrimLeft(name, "-") {
		if match(platform.Os, sysinfo.OSName()) &&
			match(platform.Version, sysinfo.OSVersion()) &&
			match(platform.Architecture, sysinfo.OSArchitecture()) &&
			match(platform.Libc, sysinfo.OSLibc()) &&
			match(platform.Compiler, sysinfo.OSCompiler()) {
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
