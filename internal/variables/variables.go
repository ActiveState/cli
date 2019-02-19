package variables

import (
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var (
	// FailExpandVariable identifies a failure during variable expansion.
	FailExpandVariable = failures.Type("variables.fail.expandvariable", failures.FailUser)

	// FailExpandVariableBadCategory identifies a variable expansion failure due to a bad variable category.
	FailExpandVariableBadCategory = failures.Type("variables.fail.expandvariable.badcategory", FailExpandVariable)

	// FailExpandVariableBadName identifies a variable expansion failure due to a bad variable name.
	FailExpandVariableBadName = failures.Type("variables.fail.expandvariable.badName", FailExpandVariable)

	// FailExpandVariableRecursion identifies a variable expansion failure due to infinite recursion.
	FailExpandVariableRecursion = failures.Type("variables.fail.expandvariable.recursion", FailExpandVariable)

	// FailExpanderBadName is used when an Expanders name is invalid.
	FailExpanderBadName = failures.Type("variables.fail.expander.badName", failures.FailVerify)

	// FailExpanderNoFunc is used when no handler func is found for an Expander.
	FailExpanderNoFunc = failures.Type("variables.fail.expander.noFunc", failures.FailVerify)
)

var lastFailure *failures.Failure

var calls int // for preventing infinite recursion during recursively expansion

// Failure retrieves the latest failure
func Failure() *failures.Failure {
	return lastFailure
}

// Expand will detect the active project and invoke ExpandFromProject with the given string
func Expand(s string) string {
	return ExpandFromProject(s, projectfile.Get())
}

// ExpandFromProject searches for $category.name-style variables in the given
// string and substitutes them with their contents, derived from the given
// project, and subject to the given constraints (if any).
func ExpandFromProject(s string, p *projectfile.Project) string {
	lastFailure = nil

	calls++
	if calls > 10 {
		calls = 0 // reset
		lastFailure = FailExpandVariableRecursion.New("error_expand_variable_infinite_recursion", s)
		print.Warning(lastFailure.Error())
		return ""
	}
	regex := regexp.MustCompile("\\${?\\w+\\.[\\w-]+}?")
	expanded := regex.ReplaceAllStringFunc(s, func(variable string) string {
		components := strings.Split(strings.Trim(variable, "${}"), ".")
		category, name := components[0], components[1]
		var value string

		if expanderFn, foundExpander := expanderRegistry[category]; foundExpander {
			var failure *failures.Failure

			if value, failure = expanderFn(name, p); failure != nil {
				lastFailure = FailExpandVariableBadName.New("error_expand_variable_project_unknown_name", variable, failure.Error())
				print.Warning(lastFailure.Error())
			}
		} else {
			lastFailure = FailExpandVariableBadCategory.New("error_expand_variable_project_unknown_category", variable, category)
			print.Warning(lastFailure.Error())
		}

		if value != "" {
			value = ExpandFromProject(value, p)
		}
		return value
	})
	calls--
	return expanded
}

// ExpanderFunc defines a function which can expand the name for a category. An Expander expects the name
// to be expanded along with the project-file definition. It will return the expanded value of the name
// or a Failure if expansion was unsuccessful.
type ExpanderFunc func(name string, project *projectfile.Project) (string, *failures.Failure)

// expanderRegistry maps category names to their ExpanderFunc implementations.
var expanderRegistry = map[string]ExpanderFunc{
	"platform":  PlatformExpander,
	"variables": VariableExpander,
	"events":    EventExpander,
	"scripts":   ScriptExpander,
}

// RegisterExpander registers an ExpanderFunc for some given handler value. The handler value must not
// effectively be a blank string and the ExpanderFunc must be defined. It is definitely possible to
// replace an existing handler using this function.
func RegisterExpander(handle string, expanderFn ExpanderFunc) *failures.Failure {
	cleanHandle := strings.TrimSpace(handle)
	if cleanHandle == "" {
		return FailExpanderBadName.New("variables_expander_err_empty_name")
	} else if expanderFn == nil {
		return FailExpanderNoFunc.New("variables_expander_err_undefined")
	}
	expanderRegistry[cleanHandle] = expanderFn
	return nil
}

// PlatformExpander expends metadata about the current platform.
func PlatformExpander(name string, project *projectfile.Project) (string, *failures.Failure) {
	for _, platform := range project.Platforms {
		if !constraints.PlatformMatches(platform) {
			continue
		}

		switch name {
		case "name":
			return platform.Name, nil
		case "os":
			return platform.Os, nil
		case "version":
			return platform.Version, nil
		case "architecture":
			return platform.Architecture, nil
		case "libc":
			return platform.Libc, nil
		case "compiler":
			return platform.Compiler, nil
		default:
			return "", FailExpandVariableBadName.New("error_expand_variable_project_unrecognized_platform_var", name)
		}
	}
	return "", nil
}

// VariableExpander expands variables defined in the profect-file.
func VariableExpander(name string, project *projectfile.Project) (string, *failures.Failure) {
	var value string
	for _, variable := range project.Variables {
		if variable.Name == name && !constraints.IsConstrained(variable.Constraints) {
			value = variable.Value
			break
		}
	}
	if value == "" {
		// Read from config file or prompt the user for a value.
		value = ConfigValue(name, project.Path())
	}
	return value, nil
}

// EventExpander expands events defined in the project-file.
func EventExpander(name string, project *projectfile.Project) (string, *failures.Failure) {
	var value string
	for _, event := range project.Events {
		if event.Name == name && !constraints.IsConstrained(event.Constraints) {
			value = event.Value
			break
		}
	}
	return value, nil
}

// ScriptExpander expands scripts defined in the project-file.
func ScriptExpander(name string, project *projectfile.Project) (string, *failures.Failure) {
	var value string
	for _, script := range project.Scripts {
		if script.Name == name && !constraints.IsConstrained(script.Constraints) {
			value = script.Value
			break
		}
	}
	return value, nil
}
