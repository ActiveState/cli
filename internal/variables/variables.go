package variables

import (
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// FailExpandVariable identifies a failure during variable expansion.
var FailExpandVariable = failures.Type("variables.fail.expandvariable", failures.FailUser)

// FailExpandVariableBadCategory identifies a variable expansion failure due to a bad variable category.
var FailExpandVariableBadCategory = failures.Type("variables.fail.expandvariable.badcategory", FailExpandVariable)

// FailExpandVariableBadName identifies a variable expansion failure due to a bad variable name.
var FailExpandVariableBadName = failures.Type("variables.fail.expandvariable.badName", FailExpandVariable)

// FailExpandVariableRecursion identifies a variable expansion failure due to infinite recursion.
var FailExpandVariableRecursion = failures.Type("variables.fail.expandvariable.recursion", FailExpandVariable)

var calls int // for preventing infinite recursion during recursively expansion

// Expand will detect the active project and invoke ExpandFromProject with the given string
func Expand(s string) (string, *failures.Failure) {
	return ExpandFromProject(s, projectfile.Get())
}

// ExpandFromProject searches for $category.name-style variables in the given
// string and substitutes them with their contents, derived from the given
// project, and subject to the given constraints (if any).
func ExpandFromProject(s string, p *projectfile.Project) (string, *failures.Failure) {
	calls++
	if calls > 10 {
		calls = 0 // reset
		return "", FailExpandVariableRecursion.New("error_expand_variable_infinite_recursion", s)
	}
	var failure *failures.Failure
	regex := regexp.MustCompile("\\${?\\w+\\.\\w+}?")
	expanded := regex.ReplaceAllStringFunc(s, func(variable string) string {
		components := strings.Split(strings.Trim(variable, "${}"), ".")
		category, name := components[0], components[1]
		var value string
		switch category {
		case "platform":
			for _, platform := range p.Platforms {
				if !constraints.PlatformMatches(platform) {
					continue
				}
				switch name {
				case "name":
					value = platform.Name
				case "os":
					value = platform.Os
				case "version":
					value = platform.Version
				case "architecture":
					value = platform.Architecture
				case "libc":
					value = platform.Libc
				case "compiler":
					value = platform.Compiler
				default:
					failure = FailExpandVariableBadName.New("error_expand_variable_project_unknown_name", variable, name)
				}
			}
		case "variables":
			for _, variable := range p.Variables {
				if variable.Name == name && !constraints.IsConstrained(variable.Constraints) {
					value = variable.Value
					break
				}
			}
			if value == "" {
				// Read from config file or prompt the user for a value.
				value = ConfigValue(name, p.Path())
			}
		case "hooks":
			for _, hook := range p.Hooks {
				if hook.Name == name && !constraints.IsConstrained(hook.Constraints) {
					value = hook.Value
					break
				}
			}
		case "commands":
			for _, command := range p.Commands {
				if command.Name == name && !constraints.IsConstrained(command.Constraints) {
					value = command.Value
					break
				}
			}
		default:
			failure = FailExpandVariableBadCategory.New("error_expand_variable_project_unknown_category", variable, category)
		}
		if value != "" {
			value, failure = ExpandFromProject(value, p)
		}
		return value
	})
	calls--
	return expanded, failure
}
