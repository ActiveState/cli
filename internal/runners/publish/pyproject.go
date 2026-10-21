package publish

import (
	"cmp"
	"regexp"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// pythonRequirement is a PEP 508 requirement translated into the platform's
// name, version-requirements, and conditions form.
type pythonRequirement struct {
	Name                string
	VersionRequirements string
	Conditions          []request.Dependency
}

var (
	// requirementRegexp matches `name[extras] specifiers`, where the specifiers
	// may be wrapped in parentheses (the legacy `name (>=1.0)` form).
	requirementRegexp = regexp.MustCompile(`^([A-Za-z0-9](?:[A-Za-z0-9._-]*[A-Za-z0-9])?)\s*(?:\[[^\]]*\])?\s*\(?([^()]*)\)?$`)
	specifierRegexp   = regexp.MustCompile(`^(~=|===|==|!=|<=|>=|<|>)\s*(\S+)$`)
	markerRegexp      = regexp.MustCompile(`^(\w+)\s*(===|==|!=|<=|>=|~=|<|>)\s*["']([^"']*)["']$`)
)

// markerKernels maps the platform markers we support to the platform's kernel names.
var markerKernels = map[string]map[string]string{
	"sys_platform":    {"win32": "Windows", "linux": "Linux", "darwin": "Darwin"},
	"platform_system": {"Windows": "Windows", "Linux": "Linux", "Darwin": "Darwin"},
	"os_name":         {"nt": "Windows"},
}

// parsePythonRequirement translates a PEP 508 requirement string. Operating
// system and Python version markers become conditions. Other markers, direct URL
// references, and the != and === operators have no platform equivalent and are
// rejected, so the caller can tell the user to add them with --depend instead.
// Extras are dropped.
func parsePythonRequirement(req string) (*pythonRequirement, error) {
	req, marker, _ := strings.Cut(strings.TrimSpace(req), ";")
	req = strings.TrimSpace(req)
	conditions, err := translateMarker(strings.TrimSpace(marker))
	if err != nil {
		return nil, err
	}
	if strings.Contains(req, "@") {
		return nil, errs.New("direct URL references are not supported")
	}
	m := requirementRegexp.FindStringSubmatch(req)
	if m == nil {
		return nil, errs.New("not a valid requirement")
	}

	var constraints []string
	for _, spec := range strings.Split(m[2], ",") {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}
		sm := specifierRegexp.FindStringSubmatch(spec)
		if sm == nil {
			return nil, errs.New("invalid version specifier %q", spec)
		}
		translated, err := translateSpecifier(sm[1], sm[2])
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, translated...)
	}

	return &pythonRequirement{Name: m[1], VersionRequirements: strings.Join(constraints, ","), Conditions: conditions}, nil
}

// translateMarker turns an environment marker made of `and`-joined operating
// system and Python version clauses into platform conditions, which the
// platform also requires all of. Python version clauses are merged into a
// single condition.
func translateMarker(marker string) ([]request.Dependency, error) {
	if marker == "" {
		return nil, nil
	}
	if strings.Contains(marker, " or ") || strings.ContainsAny(marker, "()") {
		return nil, errs.New("only 'and' is supported in environment markers")
	}

	var conditions []request.Dependency
	var pythonReqs []string
	for _, clause := range strings.Split(marker, " and ") {
		clause = strings.TrimSpace(clause)
		m := markerRegexp.FindStringSubmatch(clause)
		if m == nil {
			return nil, errs.New("unsupported environment marker '%s'", clause)
		}
		variable, op, value := m[1], m[2], m[3]

		switch variable {
		case "python_version", "python_full_version":
			reqs, err := translatePythonMarker(variable, op, value)
			if err != nil {
				return nil, err
			}
			pythonReqs = append(pythonReqs, reqs...)
		case "extra":
			return nil, errs.New("optional dependencies (extras) are not published")
		default:
			kernels, ok := markerKernels[variable]
			if !ok {
				return nil, errs.New("unsupported environment marker '%s'", clause)
			}
			kernel, ok := kernels[value]
			if !ok || op != "==" {
				return nil, errs.New("unsupported environment marker '%s'", clause)
			}
			conditions = append(conditions, request.Dependency{Name: kernel, Namespace: "kernel", VersionRequirements: ">=0"})
		}
	}

	if len(pythonReqs) > 0 {
		conditions = append(conditions, request.Dependency{
			Name:                "python",
			Namespace:           model.NewNamespaceLanguage().String(),
			VersionRequirements: strings.Join(pythonReqs, ","),
		})
	}
	return conditions, nil
}

// translatePythonMarker turns a Python version marker clause into version
// constraints. python_version holds only major.minor, so comparisons against it
// are widened to cover every patch release: python_version > "3.9" excludes
// 3.9.18, so it becomes >=3.10.
func translatePythonMarker(variable, op, version string) ([]string, error) {
	if op == "!=" || op == "===" || op == "~=" || strings.Contains(version, "*") {
		return nil, errs.New("unsupported environment marker '%s %s \"%s\"'", variable, op, version)
	}
	if variable == "python_full_version" {
		return []string{op + version}, nil
	}

	next, err := bumpRelease(strings.Split(version, "."))
	if err != nil {
		return nil, err
	}
	switch op {
	case ">":
		return []string{">=" + next}, nil
	case "<=":
		return []string{"<" + next}, nil
	case "==":
		return []string{">=" + version, "<" + next}, nil
	}
	return []string{op + version}, nil // >= and < already align with release boundaries
}

// translateSpecifier turns one PEP 440 specifier into platform constraints,
// expanding the compatible-release (~=) and prefix-match (==X.*) forms into
// explicit ranges.
func translateSpecifier(op, version string) ([]string, error) {
	switch op {
	case ">=", "<=", ">", "<":
		return []string{op + version}, nil
	case "==":
		if prefix, ok := strings.CutSuffix(version, ".*"); ok {
			upper, err := bumpRelease(strings.Split(prefix, "."))
			if err != nil {
				return nil, err
			}
			return []string{">=" + prefix, "<" + upper}, nil
		}
		return []string{"==" + version}, nil
	case "~=":
		parts := strings.Split(version, ".")
		if len(parts) < 2 {
			return nil, errs.New("~= requires at least two version components: %q", version)
		}
		upper, err := bumpRelease(parts[:len(parts)-1])
		if err != nil {
			return nil, err
		}
		return []string{">=" + version, "<" + upper}, nil
	}
	return nil, errs.New("the %s operator is not supported", op)
}

// bumpRelease increments the last release component: [1 4] -> "1.5".
func bumpRelease(parts []string) (string, error) {
	last, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return "", errs.New("cannot compute an upper bound for version %q", strings.Join(parts, "."))
	}
	bumped := append(append([]string{}, parts[:len(parts)-1]...), strconv.Itoa(last+1))
	return strings.Join(bumped, "."), nil
}

// pyprojectDependencies translates pyproject.toml dependencies into runtime
// dependencies on the platform's Python packages. Requirements that cannot be
// translated are skipped with a notice so the user can add them with --depend.
func (r *Runner) pyprojectDependencies(requires []string) ([]request.PublishVariableDep, error) {
	var reqs []*pythonRequirement
	var names []string
	for _, raw := range requires {
		req, err := parsePythonRequirement(raw)
		if err != nil {
			r.out.Notice(locale.Tl("publish_pyproject_dep_skipped",
				"Skipping pyproject.toml dependency '[ACTIONABLE]{{.V0}}[/RESET]' ({{.V1}}). Add it with '[ACTIONABLE]--depend[/RESET]' if it is required.",
				raw, err.Error()))
			continue
		}
		reqs = append(reqs, req)
		names = append(names, req.Name)
	}
	if len(reqs) == 0 {
		return nil, nil
	}

	ns := model.NewNamespacePackage("python")
	normalized, err := model.FetchNormalizedNames(ns, names, r.auth)
	if err != nil {
		return nil, locale.WrapError(err, "err_publish_pyproject_normalize", "Could not resolve the pyproject.toml dependency names")
	}

	deps := make([]request.PublishVariableDep, 0, len(reqs))
	for _, req := range reqs {
		deps = append(deps, request.PublishVariableDep{
			Dependency: request.Dependency{
				Name:                cmp.Or(normalized[req.Name], req.Name),
				Namespace:           ns.String(),
				VersionRequirements: req.VersionRequirements,
				Type:                request.DependencyTypeRuntime,
			},
			Conditions: req.Conditions,
		})
	}
	return deps, nil
}
