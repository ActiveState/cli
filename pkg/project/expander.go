package project

import (
	"regexp"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/rxutils"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/failures"
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

// Expand will detect the active project and invoke ExpandFromProject with the given string
func Expand(s string, out output.Outputer, prompt prompt.Prompter) (string, error) {
	return ExpandFromProject(s, Get())
}

// ExpandFromProject searches for $category.name-style variables in the given
// string and substitutes them with their contents, derived from the given
// project, and subject to the given constraints (if any).
func ExpandFromProject(s string, p *Project) (string, error) {
	return limitExpandFromProject(0, s, p)
}

// limitExpandFromProject limits the depth of an expansion to avoid infinite expansion of a value.
func limitExpandFromProject(depth int, s string, p *Project) (string, error) {
	if depth > constants.ExpanderMaxDepth {
		return "", locale.NewInputError("err_expand_recursion", "Infinite recursion trying to expand variable '{{.V0}}'", s)
	}

	regex := regexp.MustCompile("\\${?(\\w+)\\.([\\w-]+)+\\.?([\\w-]+)?(\\(\\))?}?")
	var err error
	expanded := rxutils.ReplaceAllStringSubmatchFunc(regex, s, func(groups []string) string {
		if err != nil {
			return ""
		}
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
			var err2 error
			if value, err2 = expanderFn(name, meta, isFunction, p); err2 != nil {
				err = errs.Wrap(err2, "Could not expand %s.%s", category, name)
				return ""
			}
		} else {
			err = locale.NewInputError("err_expand_category", "Error expanding variable '{{.V0}}': unknown category '{{.V1}}'", variable, category)
			return ""
		}

		if value != "" {
			value, err = limitExpandFromProject(depth+1, value, p)
		}
		return value
	})

	return expanded, err
}

// ExpanderFunc defines an Expander function which can expand the name for a category. An Expander expects the name
// to be expanded along with the project-file definition. It will return the expanded value of the name
// or a Failure if expansion was unsuccessful.
type ExpanderFunc func(name string, meta string, isFunction bool, project *Project) (string, error)

// PlatformExpander expends metadata about the current platform.
func PlatformExpander(name string, meta string, isFunction bool, project *Project) (string, error) {
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
			return "", locale.NewInputError("err_expand_platform", "Unrecognized platform variable '{{.V0}}'", name)
		}
	}
	return "", nil
}

// EventExpander expands events defined in the project-file.
func EventExpander(name string, meta string, isFunction bool, project *Project) (string, error) {
	projectFile := project.Source()
	constrained, err := constraints.FilterUnconstrained(pConditional, projectFile.Events.AsConstrainedEntities())
	if err != nil {
		return "", err
	}
	for _, v := range constrained {
		if v.ID() == name {
			return projectfile.MakeEventsFromConstrainedEntities([]projectfile.ConstrainedEntity{v})[0].Value, nil
		}
	}
	return "", nil
}

// ScriptExpander expands scripts defined in the project-file.
func ScriptExpander(name string, meta string, isFunction bool, project *Project) (string, error) {
	script := project.ScriptByName(name)
	if script == nil {
		return "", nil
	}

	if meta == "path" && isFunction {
		return expandPath(name, script)
	}
	return script.Raw(), nil
}

func expandPath(name string, script *Script) (string, error) {
	if script.cachedFile() != "" {
		return script.cachedFile(), nil
	}

	sf, fail := scriptfile.NewEmpty(script.LanguageSafe(), name)
	if fail != nil {
		return "", fail.ToError()
	}
	script.setCachedFile(sf.Filename())

	v, err := script.Value()
	if err != nil {
		return "", err
	}
	fail = sf.Write(v)
	if fail != nil {
		return "", fail.ToError()
	}

	return sf.Filename(), nil
}

// ConstantExpander expands constants defined in the project-file.
func ConstantExpander(name string, meta string, isFunction bool, project *Project) (string, error) {
	projectFile := project.Source()
	constrained, err := constraints.FilterUnconstrained(pConditional, projectFile.Constants.AsConstrainedEntities())
	if err != nil {
		return "", err
	}
	for _, v := range constrained {
		if v.ID() == name {
			return projectfile.MakeConstantsFromConstrainedEntities([]projectfile.ConstrainedEntity{v})[0].Value, nil
		}
	}
	return "", nil
}
