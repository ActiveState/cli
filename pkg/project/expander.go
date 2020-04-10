package project

import (
	"regexp"

	"github.com/ActiveState/cli/internal/scriptfile"

	"github.com/ActiveState/cli/internal/rxutils"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
)

var (
	// FailExpandVariable identifies a failure during variable expansion.
	FailExpandVariable = failures.Type("project.fail.expandvariable", failures.FailUser)

	// FailExpandVariableBadCategory identifies a variable expansion failure due to a bad variable category.
	FailExpandVariableBadCategory = failures.Type("project.fail.expandvariable.badcategory", FailExpandVariable)

	// FailExpandVariableBadName identifies a variable expansion failure due to a bad variable name.
	FailExpandVariableBadName = failures.Type("project.fail.expandvariable.badName", FailExpandVariable)

	// FailExpandVariableRecursion identifies a variable expansion failure due to infinite recursion.
	FailExpandVariableRecursion = failures.Type("project.fail.expandvariable.recursion", FailExpandVariable)

	// FailExpanderBadName is used when an Expanders name is invalid.
	FailExpanderBadName = failures.Type("project.fail.expander.badName", failures.FailVerify)

	// FailExpanderNoFunc is used when no handler func is found for an Expander.
	FailExpanderNoFunc = failures.Type("project.fail.expander.noFunc", failures.FailVerify)

	// FailVarNotFound is used when no handler func is found for an Expander.
	FailVarNotFound = failures.Type("project.fail.vars.notfound", FailExpandVariable)
)

var lastFailure *failures.Failure

// Failure retrieves the latest failure
func Failure() *failures.Failure {
	return lastFailure
}

// Expand will detect the active project and invoke ExpandFromProject with the given string
func Expand(s string) string {
	return ExpandFromProject(s, Get())
}

// Prompter is accessible so tests can overwrite it with Mock.  Do not use if you're not writing code for this package
var Prompter prompt.Prompter

func init() {
	Prompter = prompt.New()
}

// ExpandFromProject searches for $category.name-style variables in the given
// string and substitutes them with their contents, derived from the given
// project, and subject to the given constraints (if any).
func ExpandFromProject(s string, p *Project) string {
	return limitExpandFromProject(0, s, p)
}

// limitExpandFromProject limits the depth of an expansion to avoid infinite expansion of a value.
func limitExpandFromProject(depth int, s string, p *Project) string {
	lastFailure = nil
	if depth > constants.ExpanderMaxDepth {
		lastFailure = FailExpandVariableRecursion.New("error_expand_variable_infinite_recursion", s)
		print.Warning(lastFailure.Error())
		return ""
	}

	regex := regexp.MustCompile("\\${?(\\w+)\\.([\\w-]+)+\\.?([\\w-]+)?(\\(\\))?}?")
	expanded := rxutils.ReplaceAllStringSubmatchFunc(regex, s, func(groups []string) string {
		var variable, category, name, meta string
		var isFunction bool

		variable = groups[0]
		category = groups[1]
		name = groups[2]
		if len(groups) > 3 {
			meta = groups[3]
		}
		if len(groups) > 4 {
			isFunction = true
		}

		var value string

		if expanderFn, foundExpander := expanderRegistry[category]; foundExpander {
			var failure *failures.Failure

			if value, failure = expanderFn(name, meta, isFunction, p); failure != nil {
				lastFailure = FailExpandVariableBadName.New("error_expand_variable_project_unknown_name", variable, failure.Error())
				print.Warning(lastFailure.Error())
			}
		} else {
			lastFailure = FailExpandVariableBadCategory.New("error_expand_variable_project_unknown_category", variable, category)
			print.Warning(lastFailure.Error())
		}

		if value != "" {
			value = limitExpandFromProject(depth+1, value, p)
		}
		return value
	})

	return expanded
}

// ExpanderFunc defines an Expander function which can expand the name for a category. An Expander expects the name
// to be expanded along with the project-file definition. It will return the expanded value of the name
// or a Failure if expansion was unsuccessful.
type ExpanderFunc func(name string, meta string, isFunction bool, project *Project) (string, *failures.Failure)

// PlatformExpander expends metadata about the current platform.
func PlatformExpander(name string, meta string, isFunction bool, project *Project) (string, *failures.Failure) {
	projectFile := project.Source()
	for _, platform := range projectFile.Platforms {
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

// EventExpander expands events defined in the project-file.
func EventExpander(name string, meta string, isFunction bool, project *Project) (string, *failures.Failure) {
	projectFile := project.Source()

	i, ev := constraints.MostSpecificUnconstrainedEvent(name, projectFile.Events)
	if i < 0 {
		return "", nil
	}
	return ev.Value, nil
}

// ScriptExpander expands scripts defined in the project-file.
func ScriptExpander(name string, meta string, isFunction bool, project *Project) (string, *failures.Failure) {
	script := project.ScriptByName(name)
	if script == nil {
		return "", nil
	}

	if meta == "path" && isFunction {
		return expandPath(name, script)
	}
	return script.Raw(), nil
}

func expandPath(name string, script *Script) (string, *failures.Failure) {
	if script.cachedFile() != "" {
		return script.cachedFile(), nil
	}

	sf, fail := scriptfile.NewEmpty(script.LanguageSafe(), name)
	if fail != nil {
		return "", fail
	}
	script.setCachedFile(sf.Filename())

	fail = sf.Write(script.Value())
	if fail != nil {
		return "", fail
	}

	return sf.Filename(), nil
}

// ConstantExpander expands constants defined in the project-file.
func ConstantExpander(name string, meta string, isFunction bool, project *Project) (string, *failures.Failure) {
	projectFile := project.Source()
	i, c := constraints.MostSpecificUnconstrainedConstant(name, projectFile.Constants)
	if i < 0 {
		return "", nil
	}
	return c.Value, nil
}
