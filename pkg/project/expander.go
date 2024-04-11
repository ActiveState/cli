package project

import (
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/rxutils"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Expansion struct {
	Project      *Project
	Script       *Script
	BashifyPaths bool
}

func NewExpansion(p *Project) *Expansion {
	return &Expansion{Project: p}
}

// ApplyWithMaxDepth limits the depth of an expansion to avoid infinite expansion of a value.
func (ctx *Expansion) ApplyWithMaxDepth(s string, depth int) (string, error) {
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
		lastGroup := groups[len(groups)-1]
		if strings.HasPrefix(lastGroup, "(") && strings.HasSuffix(lastGroup, ")") {
			isFunction = true
		}

		var value string

		if expanderFn, foundExpander := expanderRegistry[category]; foundExpander {
			var err2 error
			if value, err2 = expanderFn(variable, name, meta, isFunction, ctx); err2 != nil {
				err = errs.Wrap(err2, "Could not expand %s.%s", category, name)
				return ""
			}
		} else {
			return variable // we don't control this variable, so leave it as is
		}

		if value != "" && value != variable {
			value, err = ctx.ApplyWithMaxDepth(value, depth+1)
		}
		return value
	})

	return expanded, err
}

// ExpandFromProject searches for $category.name-style variables in the given
// string and substitutes them with their contents, derived from the given
// project, and subject to the given constraints (if any).
func ExpandFromProject(s string, p *Project) (string, error) {
	return NewExpansion(p).ApplyWithMaxDepth(s, 0)
}

// ExpandFromProjectBashifyPaths is like ExpandFromProject, but bashifies all instances of
// $script.name.path().
func ExpandFromProjectBashifyPaths(s string, p *Project) (string, error) {
	expansion := &Expansion{Project: p, BashifyPaths: true}
	return expansion.ApplyWithMaxDepth(s, 0)
}

func ExpandFromScript(s string, script *Script) (string, error) {
	expansion := &Expansion{
		Project:      script.project,
		Script:       script,
		BashifyPaths: runtime.GOOS == "windows" && (script.LanguageSafe()[0] == language.Bash || script.LanguageSafe()[0] == language.Sh),
	}
	return expansion.ApplyWithMaxDepth(s, 0)
}

// ExpanderFunc defines an Expander function which can expand the name for a category. An Expander expects the name
// to be expanded along with the project-file definition. It will return the expanded value of the name
// or a Failure if expansion was unsuccessful.
type ExpanderFunc func(variable, name, meta string, isFunction bool, ctx *Expansion) (string, error)

// EventExpander expands events defined in the project-file.
func EventExpander(_ string, name string, meta string, isFunction bool, ctx *Expansion) (string, error) {
	projectFile := ctx.Project.Source()
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
func ScriptExpander(_ string, name string, meta string, isFunction bool, ctx *Expansion) (string, error) {
	script := ctx.Project.ScriptByName(name)
	if script == nil {
		return "", nil
	}

	if !isFunction {
		return script.Raw(), nil
	}

	if meta == "path" || meta == "path._posix" {
		path, err := expandPath(name, script)
		if err != nil {
			return "", err
		}

		if ctx.BashifyPaths || meta == "path._posix" {
			return osutils.BashifyPath(path)
		}

		return path, nil
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
func (m *Mixin) Expander(_ string, name string, meta string, _ bool, _ *Expansion) (string, error) {
	if name == "user" {
		return userExpander(m.auth, meta), nil
	}
	return "", nil
}

// ConstantExpander expands constants defined in the project-file.
func ConstantExpander(_ string, name string, meta string, isFunction bool, ctx *Expansion) (string, error) {
	projectFile := ctx.Project.Source()
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

// ProjectExpander expands constants defined in the project-file.
func ProjectExpander(_ string, name string, _ string, isFunction bool, ctx *Expansion) (string, error) {
	if !isFunction {
		return "", nil
	}

	project := ctx.Project
	switch name {
	case "url":
		return project.URL(), nil
	case "commit":
		commitID := project.LegacyCommitID() // Not using localcommit due to import cycle. See anti-pattern comment in localcommit pkg.
		return commitID, nil
	case "branch":
		return project.BranchName(), nil
	case "owner":
		return project.Namespace().Owner, nil
	case "name":
		return project.Namespace().Project, nil
	case "namespace":
		return project.Namespace().String(), nil
	case "path":
		path := project.Source().Path()
		if path == "" {
			return path, nil
		}
		dir := filepath.Dir(path)
		if ctx.BashifyPaths {
			return osutils.BashifyPath(dir)
		}
		return dir, nil
	}

	return "", nil
}

func TopLevelExpander(variable string, name string, _ string, _ bool, ctx *Expansion) (string, error) {
	projectFile := ctx.Project.Source()
	switch name {
	case "project":
		return projectFile.Project, nil
	case "lock":
		return projectFile.Lock, nil
	}
	return variable, nil
}
