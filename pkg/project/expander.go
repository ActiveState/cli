package project

import (
	"regexp"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/rxutils"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
)

// Expand will detect the active project and invoke ExpandFromProject with the given string
func Expand(s string) (string, error) {
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

	regex := regexp.MustCompile(`\${?(\w+)\.?([\w-]+)?\.?([\w\.-]+)?(\(\))?}?`)
	var err error
	expanded := rxutils.ReplaceAllStringSubmatchFunc(regex, s, func(groups []string) string {
		if err != nil {
			return ""
		}
		var variable, category, name, meta string
		var isFunction bool
		variable = groups[0]

		if len(groups) == 2 {
			category = "toplevel"
			name = groups[1]
		}
		if len(groups) > 2 {
			category = groups[1]
			name = groups[2]
		}
		if len(groups) > 3 {
			meta = groups[3]
		}
		if len(groups) > 4 {
			isFunction = true
		}

		var value string

		if expanderFn, foundExpander := expanderRegistry[category]; foundExpander {
			var err2 error
			if value, err2 = expanderFn(variable, name, meta, isFunction, p); err2 != nil {
				err = errs.Wrap(err2, "Could not expand %s.%s", category, name)
				return ""
			}
		} else {
			return variable // we don't control this variable, so leave it as is
		}

		if value != "" && value != variable {
			value, err = limitExpandFromProject(depth+1, value, p)
		}
		return value
	})

	return expanded, err
}

// ExpanderFunc defines an Expander function which can expand the name for a category. An Expander expects the name
// to be expanded along with the project-file definition. It will return the expanded value of the name
// or a Failure if expansion was unsuccessful.
type ExpanderFunc func(variable, name, meta string, isFunction bool, project *Project) (string, error)

// PlatformExpander expends metadata about the current platform.
func PlatformExpander(_ string, name string, meta string, isFunction bool, project *Project) (string, error) {
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
func EventExpander(_ string, name string, meta string, isFunction bool, project *Project) (string, error) {
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
func ScriptExpander(_ string, name string, meta string, isFunction bool, project *Project) (string, error) {
	script := project.ScriptByName(name)
	if script == nil {
		return "", nil
	}

	if !isFunction {
		return script.Raw(), nil
	}

	switch meta {
	case "path":
		return expandPath(name, script)
	case "path.posix":
		path, err := expandPath(name, script)
		if err != nil {
			return "", err
		}
		return osutils.BashifyPath(path)
	}

	return script.Raw(), nil
}

func expandPath(name string, script *Script) (string, error) {
	if script.cachedFile() != "" {
		return script.cachedFile(), nil
	}

	languages := script.LanguageSafe()
	if len(languages) == 0 {
		languages = DefaultScriptLanguage()
	}

	sf, err := scriptfile.NewEmpty(languages[0], name)
	if err != nil {
		return "", err
	}
	script.setCachedFile(sf.Filename())

	v, err := script.Value()
	if err != nil {
		return "", err
	}
	err = sf.Write(v)
	if err != nil {
		return "", err
	}

	return sf.Filename(), nil
}

// userExpander
func userExpander(auth *authentication.Auth, element string) string {
	if element == "name" {
		return auth.WhoAmI()
	}
	if element == "email" {
		return auth.Email()
	}
	if element == "jwt" {
		return auth.BearerToken()
	}
	return ""
}

// Mixin provides expansions that are not sourced from a project file
type Mixin struct {
	auth *authentication.Auth
}

// NewMixin creates a Mixin object providing extra expansions
func NewMixin(auth *authentication.Auth) *Mixin {
	return &Mixin{auth}
}

// Expander expands mixin variables
func (m *Mixin) Expander(_ string, name string, meta string, _ bool, _ *Project) (string, error) {
	if name == "user" {
		return userExpander(m.auth, meta), nil
	}
	return "", nil
}

// ConstantExpander expands constants defined in the project-file.
func ConstantExpander(_ string, name string, meta string, isFunction bool, project *Project) (string, error) {
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

func TopLevelExpander(variable string, name string, _ string, _ bool, project *Project) (string, error) {
	projectFile := project.Source()
	switch name {
	case "project":
		return projectFile.Project, nil
	case "lock":
		return projectFile.Lock, nil
	}
	return variable, nil
}
