package variables

import (
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var calls int // for preventing infinite recursion during recursively expansion

// ExpandFromProject searches for $category.name-style variables in the given
// string and substitutes them with their contents, derived from the given
// project, and subject to the given constraints (if any).
func ExpandFromProject(s string, p *projectfile.Project) (string, *failures.Failure) {
	calls++
	if calls > 10 {
		calls = 0 // reset
		return "", failures.FailExpandVariableRecursion.New(locale.T("error_expand_variable_infinite_recursion", map[string]string{"Variable": s}))
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
					failure = failures.FailExpandVariableBadName.New(locale.T("error_expand_variable_project_unknown_name", map[string]string{
						"Variable": variable,
						"Name":     name,
					}))
				}
			}
		case "variables":
			for _, variable := range p.Variables {
				if variable.Name == name && !constraints.IsConstrained(variable.Constraints) {
					value = variable.Value
					break
				}
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
			failure = failures.FailExpandVariableBadCategory.New(locale.T("error_expand_variable_project_unknown_category", map[string]string{
				"Variable": variable,
				"Category": category,
			}))
		}
		if value != "" {
			value, failure = ExpandFromProject(value, p)
		}
		return value
	})
	calls--
	return expanded, failure
}
