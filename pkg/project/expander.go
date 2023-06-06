package project

import (
	"fmt"
	"reflect"
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
	"github.com/ActiveState/cli/pkg/projectfile"
)

const (
	expandStructTag    = "expand"
	expandTagOptAsFunc = "asFunc"
	expandTagOptIsPath = "isPath"
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
			category = TopLevelExpanderName
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

func TopLevelExpander(variable string, name string, _ string, _ bool, ctx *Expansion) (string, error) {
	projectFile := ctx.Project.Source()
	switch name {
	case "project":
		return projectFile.Project, nil
	case "lock":
		return projectFile.Lock, nil
	default:
		if v, ok := topLevelLookup[name]; ok {
			return v, nil
		}
	}
	return variable, nil
}

// entry manages a simple value held by a field as well as the field's metadata.
type entry struct {
	asFunc bool
	isPath bool
	value  string
}

func newEntry(tag string, val reflect.Value) entry {
	var asFunc, isPath bool

	tParts := strings.Split(tag, ",")
	if len(tParts) > 1 {
		if strings.Contains(tParts[1], expandTagOptAsFunc) {
			asFunc = true
		}
		if strings.Contains(tParts[1], expandTagOptIsPath) {
			isPath = true
		}
	}

	return entry{
		asFunc: asFunc,
		isPath: isPath,
		value:  fmt.Sprintf("%v", val.Interface()),
	}
}

func makeEntryMap(structure reflect.Value) map[string]entry {
	m := make(map[string]entry)
	fields := reflect.VisibleFields(structure.Type())

	// Work at depth 3: Vars.Struct.Struct.[Simple]
	for _, f := range fields {
		if !f.IsExported() {
			continue
		}

		d3Val := structure.FieldByIndex(f.Index)
		m[strings.ToLower(f.Name)] = newEntry(f.Tag.Get(expandStructTag), d3Val)
	}

	return m
}

func makeEntryMapMap(structure reflect.Value) map[string]map[string]entry {
	m := make(map[string]map[string]entry)
	fields := reflect.VisibleFields(structure.Type())

	// Work at depth 2: Vars.Struct.[Struct].Simple
	for _, f := range fields {
		if !f.IsExported() {
			continue
		}

		d2Val := structure.FieldByIndex(f.Index)
		if d2Val.Kind() == reflect.Ptr {
			d2Val = d2Val.Elem()
		}

		switch d2Val.Type().Kind() {
		// Convert type (to map) to express advanced control like tag handling.
		case reflect.Struct:
			m[strings.ToLower(f.Name)] = makeEntryMap(d2Val)

		// Format simple value. This is a leaf: Vars.Struct.[Simple]
		// Conform to map-map, store at zero-valued key of inner map.
		default:
			m[strings.ToLower(f.Name)] = map[string]entry{
				"": newEntry(f.Tag.Get(expandStructTag), d2Val),
			}
		}
	}

	return m
}

func makeLazyExpanderFuncFromPtrToStruct(val reflect.Value) ExpanderFunc {
	return func(v, name, meta string, isFunc bool, ctx *Expansion) (string, error) {
		iface := val.Interface()
		if u, ok := iface.(interface{ Update(*Project) }); ok {
			u.Update(ctx.Project)
		}

		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		fn := makeExpanderFuncFromMap(makeEntryMapMap(val))

		return fn(v, name, meta, isFunc, ctx)
	}
}

func makeExpanderFuncFromMap(m map[string]map[string]entry) ExpanderFunc {
	return func(v, name, meta string, isFunc bool, ctx *Expansion) (string, error) {
		if isFunc && meta == "()" {
			meta = ""
		}

		if sub, ok := m[name]; ok {
			if e, ok := sub[meta]; ok && isFunc == e.asFunc {
				value := e.value
				if ctx.BashifyPaths && e.isPath {
					return osutils.BashifyPath(value)
				}

				return value, nil
			}
		}

		return "", nil
	}
}

func makeExpanderFuncFromFunc(fn reflect.Value) ExpanderFunc {
	return func(v, name, meta string, isFunc bool, ctx *Expansion) (string, error) {
		// Call function; It should not require any arguments.
		// Work at depth 1: Vars.[FuncReturnsSomething]...
		vals := fn.Call(nil)
		if len(vals) > 1 {
			if !vals[1].IsNil() {
				return "", vals[1].Interface().(error)
			}
		}

		d1Val := vals[0]
		// deref if needed
		if d1Val.Kind() == reflect.Ptr {
			d1Val = d1Val.Elem()
		}

		switch d1Val.Kind() {
		// Convert type (to map-map) to express advanced control like tag handling.
		case reflect.Struct:
			m := makeEntryMapMap(d1Val)
			expandFromMap := makeExpanderFuncFromMap(m)
			return expandFromMap(v, name, meta, isFunc, ctx)

		// Format simple value. This is a leaf: Vars.[FuncReturnsSimple]
		default:
			return fmt.Sprintf("%v", d1Val.Interface()), nil
		}
	}
}
