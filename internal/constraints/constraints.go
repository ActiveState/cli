package constraints

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var cache = make(map[string]interface{})

func getCache(key string, getter func() (interface{}, error)) (interface{}, error) {
	if v, ok := cache[key]; ok {
		return v, nil
	}
	v, err := getter()
	if err != nil {
		return nil, err
	}
	cache[key] = v
	return v, err
}

// For testing.
var osOverride, osVersionOverride, archOverride, libcOverride, compilerOverride string

type Conditional struct {
	params map[string]interface{}
	funcs  template.FuncMap
}

func NewConditional(a *authentication.Auth) *Conditional {
	c := &Conditional{map[string]interface{}{}, map[string]interface{}{}}

	c.RegisterFunc("Mixin", func() map[string]interface{} {
		res := map[string]string{
			"Name":  "",
			"Email": "",
		}
		if a.Authenticated() {
			res["Name"] = a.WhoAmI()
			res["Email"] = a.Email()
		}
		return map[string]interface{}{
			"User": res,
		}
	})
	c.RegisterFunc("Contains", funk.Contains)
	c.RegisterFunc("HasPrefix", strings.HasPrefix)
	c.RegisterFunc("HasSuffix", strings.HasSuffix)
	c.RegisterFunc("MatchRx", func(rxv, v string) bool {
		rx, err := regexp.Compile(rxv)
		if err != nil {
			logging.Warning("Invalid Regex: %s, error: %v", rxv, err)
			return false
		}
		return rx.Match([]byte(v))
	})

	return c
}

type projectable interface {
	Owner() string
	Name() string
	NamespaceString() string
	CommitID() string
	BranchName() string
	Path() string
	URL() string
}

func NewPrimeConditional(auth *authentication.Auth, pj projectable, subshellName string) *Conditional {
	var (
		pjOwner     string
		pjName      string
		pjNamespace string
		pjURL       string
		pjCommit    string
		pjBranch    string
		pjPath      string
	)
	if !p.IsNil(pj) {
		pjOwner = pj.Owner()
		pjName = pj.Name()
		pjNamespace = pj.NamespaceString()
		pjURL = pj.URL()
		pjCommit = pj.CommitID()
		pjBranch = pj.BranchName()
		pjPath = pj.Path()
		if pjPath != "" {
			pjPath = filepath.Dir(pjPath)
		}
	}

	c := NewConditional(auth)
	c.RegisterParam("Project", map[string]string{
		"Owner":     pjOwner,
		"Name":      pjName,
		"Namespace": pjNamespace,
		"Url":       pjURL,
		"Commit":    pjCommit,
		"Branch":    pjBranch,
		"Path":      pjPath,

		// Legacy
		"NamespacePrefix": pjNamespace,
	})
	osVersion, err := sysinfo.OSVersion()
	if err != nil {
		logging.Error("Could not detect OSVersion: %v", err)
		rollbar.Error("Could not detect OSVersion: %v", err)
	}
	c.RegisterParam("OS", map[string]interface{}{
		"Name":         sysinfo.OS().String(),
		"Version":      osVersion,
		"Architecture": sysinfo.Architecture().String(),
	})
	c.RegisterParam("Shell", subshellName)

	return c
}

func (c *Conditional) RegisterFunc(name string, value interface{}) {
	c.funcs[name] = value
}

func (c *Conditional) RegisterParam(name string, value interface{}) {
	c.params[name] = value
}

func (c *Conditional) Eval(conditional string) (bool, error) {
	tpl, err := template.New("letter").Funcs(c.funcs).Parse(fmt.Sprintf(`{{if %s}}1{{end}}`, conditional))
	if err != nil {
		return false, locale.WrapInputError(err, "err_conditional", "Invalid 'if' condition: '{{.V0}}', error: '{{.V1}}'.", conditional, err.Error())
	}

	result := bytes.Buffer{}
	tpl.Execute(&result, c.params)

	return result.String() == "1", nil
}

// Returns whether or not the sysinfo-detected OS matches the given one
// (presumably the constraint).
func osMatches(os string) bool {
	name := sysinfo.OS().String()
	if osOverride != "" {
		name = osOverride
	}
	return strings.ToLower(name) == strings.ToLower(os)
}

// Returns whether or not the sysinfo-detected OS version is greater than or
// equal to the given one (presumably the constraint).
// An example version constraint is "4.1.0".
func osVersionMatches(version string) bool {
	osVersionI, osVersionErr := getCache("osVersion", func() (interface{}, error) { return sysinfo.OSVersion() })
	osVersion := osVersionI.(*sysinfo.OSVersionInfo)

	if osVersionMatchesGlobbed(osVersion.Version, version) {
		return true
	}

	if osVersionOverride != "" {
		// When writing tests, this string should be of the form:
		// [major].[minor].[micro] [os free-form name]
		osVersion = &sysinfo.OSVersionInfo{}
		fmt.Sscanf(osVersionOverride, "%d.%d.%d %s", &osVersion.Major, &osVersion.Minor, &osVersion.Micro, &osVersion.Name)
		osVersion.Version = fmt.Sprintf("%d.%d.%d", osVersion.Major, osVersion.Minor, osVersion.Micro)
		osVersionErr = nil
	}
	if osVersionErr != nil {
		return false
	}
	osVersionParts := []int{osVersion.Major, osVersion.Minor, osVersion.Micro}
	for i, part := range strings.Split(version, ".") {
		versionPart, err := strconv.Atoi(part)
		if err != nil || osVersionParts[i] < versionPart {
			return false
		} else if osVersionParts[i] > versionPart {
			// If this part is greater, skip subsequent checks.
			// e.g. If osVersion is 2.6 and version is 3.0, 3 > 2, so ignore the minor
			// check (which would have failed). If osVersion is 2.6 and version is
			// 2.5, the minors would be compared.
			return true
		}
	}
	return true
}

func osVersionMatchesGlobbed(version, globbed string) bool {
	return matchesGlobbed(version, globbed)
}

func matchesGlobbed(value, term string) bool {
	if !strings.Contains(term, "*") {
		return term == value
	}

	chunks := strings.Split(term, "*")

	var mark int
	var indexes []int
	for _, chunk := range chunks {
		if chunk == "" {
			continue
		}

		index := strings.Index(value[mark:], chunk)
		if index < 0 {
			return false
		}
		index += mark

		mark = index + len(chunk)
		indexes = append(indexes, index, mark)

	}

	for iter, index := range indexes {
		if iter == 0 {
			continue
		}

		if index < indexes[iter-1] {
			return false
		}
	}

	if chunks[0] != "" && !strings.HasPrefix(value, chunks[0]) {
		return false
	}

	if chunks[len(chunks)-1] != "" && !strings.HasSuffix(value, chunks[len(chunks)-1]) {
		return false
	}

	return true
}

// Returns whether or not the sysinfo-detected platform architecture matches the
// given one (presumably the constraint).
func archMatches(arch string) bool {
	name := sysinfo.Architecture().String()
	if archOverride != "" {
		name = archOverride
	}
	return strings.ToLower(name) == strings.ToLower(arch)
}

// Returns whether or not the name of the sysinfo-detected Libc matches the
// given one (presumably the constraint) and that its version is greater than or
// equal to the given one.
// An example Libc constraint is "glibc 2.23".
func libcMatches(libc string) bool {
	osLibcI, osLibcErr := getCache("osLibc", func() (interface{}, error) { return sysinfo.Libc() })
	osLibc := osLibcI.(*sysinfo.LibcInfo)

	if libcOverride != "" {
		osLibc = &sysinfo.LibcInfo{}
		var name string
		fmt.Sscanf(libcOverride, "%s %d.%d", &name, &osLibc.Major, &osLibc.Minor)
		name = strings.ToLower(name)
		if name == strings.ToLower(sysinfo.Glibc.String()) {
			osLibc.Name = sysinfo.Glibc
		} else if name == strings.ToLower(sysinfo.Msvcrt.String()) {
			osLibc.Name = sysinfo.Msvcrt
		} else if name == strings.ToLower(sysinfo.BsdLibc.String()) {
			osLibc.Name = sysinfo.BsdLibc
		} else {
			osLibc.Name = sysinfo.UnknownLibc
		}
		osLibcErr = nil
	}
	if osLibcErr != nil {
		return false
	}
	regex := regexp.MustCompile("^([[:alpha:]]+)\\W+(\\d+)\\D(\\d+)")
	matches := regex.FindStringSubmatch(libc)
	if len(matches) != 4 {
		return false
	}
	if strings.ToLower(matches[1]) != strings.ToLower(osLibc.Name.String()) {
		return false
	}
	osLibcParts := []int{osLibc.Major, osLibc.Minor}
	for i, part := range matches[2:] {
		versionPart, err := strconv.Atoi(part)
		if err != nil || osLibcParts[i] < versionPart {
			return false
		} else if osLibcParts[i] > versionPart {
			// If this part is greater, skip subsequent checks.
			// e.g. If osLibc is 1.9 and version is 2.0, 2 > 1, so ignore the minor
			// check (which would have failed). If osVersion is 1.9 and version is
			// 1.8, the minors would be compared.
			return true
		}
	}
	return true
}

// Returns whether or not a sysinfo-detected compiler exists whose name matches
// the given one (presumably the constraint) and that its version is greater
// than or equal to the given one.
// An example compiler constraint is "gcc 7".
func compilerMatches(compiler string) bool {
	osCompilersI, osCompilersErr := getCache("osCompiler", func() (interface{}, error) { return sysinfo.Compilers() })
	osCompilers := osCompilersI.([]*sysinfo.CompilerInfo)

	if compilerOverride != "" {
		osCompilers = []*sysinfo.CompilerInfo{&sysinfo.CompilerInfo{}}
		var name string
		fmt.Sscanf(compilerOverride, "%s %d.%d", &name, &osCompilers[0].Major, &osCompilers[0].Minor)
		name = strings.ToLower(name)
		if name == strings.ToLower(sysinfo.Gcc.String()) {
			osCompilers[0].Name = sysinfo.Gcc
		} else if name == strings.ToLower(sysinfo.Msvc.String()) {
			osCompilers[0].Name = sysinfo.Msvc
		} else if name == strings.ToLower(sysinfo.Mingw.String()) {
			osCompilers[0].Name = sysinfo.Mingw
		} else if name == strings.ToLower(sysinfo.Clang.String()) {
			osCompilers[0].Name = sysinfo.Clang
		}
		osCompilersErr = nil
	}
	if osCompilersErr != nil {
		return false
	}
	regex := regexp.MustCompile("^([[:alpha:]]+)\\W+(\\d+)\\D?(\\d*)")
	matches := regex.FindStringSubmatch(compiler)
	if len(matches) != 4 {
		return false
	}
	for _, osCompiler := range osCompilers {
		if strings.ToLower(matches[1]) != strings.ToLower(osCompiler.Name.String()) {
			continue
		}
		osCompilerParts := []int{osCompiler.Major, osCompiler.Minor}
		for i, part := range matches[2:] {
			if part == "" {
				break // ignore minor check
			}
			versionPart, err := strconv.Atoi(part)
			if err != nil || osCompilerParts[i] < versionPart {
				return false
			} else if osCompilerParts[i] > versionPart {
				// If this part is greater, skip subsequent checks.
				// e.g. If osCompiler is 7.2 and compiler is 5.0, 7 > 5, so ignore the
				// minor check (which would have failed). If osCompiler is 4.6 and
				// version is 4.4, the minors would be compared.
				return true
			}
		}
		return true
	}
	return false // no matching compilers found
}

// PlatformMatches returns whether or not the given platform matches the current
// platform, as determined by the sysinfo package.
func PlatformMatches(platform projectfile.Platform) bool {
	return (platform.Os == "" || osMatches(platform.Os)) &&
		(platform.Version == "" || osVersionMatches(platform.Version)) &&
		(platform.Architecture == "" || archMatches(platform.Architecture)) &&
		(platform.Libc == "" || libcMatches(platform.Libc)) &&
		(platform.Compiler == "" || compilerMatches(platform.Compiler))
}

//Returns whether or not the current OS is constrained by the given
// named constraints, which are defined in the given project configuration.
func osIsConstrained(constraintOSes string) bool {
	names := strings.Split(constraintOSes, ",")
	constrained := true
	for _, name := range names {
		if osMatches(strings.TrimLeft(name, "-")) {
			if strings.HasPrefix(name, "-") {
				return true
			}
			constrained = false
		}
	}
	return constrained
}

// Returns whether or not the current platform is constrained by the given
// named constraints, which are defined in the given project configuration.
func platformIsConstrained(constraintNames string) bool {
	project := projectfile.Get()
	names := strings.Split(constraintNames, ",")
	constrained := true
	for _, name := range names {
		for _, platform := range project.Platforms {
			if platform.Name == strings.TrimLeft(name, "-") && PlatformMatches(platform) {
				if strings.HasPrefix(name, "-") {
					return true
				}
				constrained = false // can't return here because an exclude might still occur
			}
		}
	}

	return constrained
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
// The second return value is for the specificity of the constraint (i.e, how
// many constraints were specified and checked)
func IsConstrained(constraint projectfile.Constraint) (bool, int) {
	if constraint.Platform == "" &&
		constraint.Environment == "" &&
		constraint.OS == "" {
		return false, 0
	}
	specificity := 0
	constrained := false
	if constraint.OS != "" {
		specificity++
		constrained = constrained || osIsConstrained(constraint.OS)
	}
	if constraint.Platform != "" {
		specificity++
		constrained = constrained || platformIsConstrained(constraint.Platform)
	}
	if constraint.Environment != "" {
		specificity++
		constrained = constrained || environmentIsConstrained(constraint.Environment)
	}
	return constrained, specificity
}

// FilterUnconstrained filters a list of constrained entities and returns only
// those which are unconstrained. If two items with the same name exist, only
// the most specific item will be added to the results.
func FilterUnconstrained(conditional *Conditional, items []projectfile.ConstrainedEntity) ([]projectfile.ConstrainedEntity, error) {
	type itemIndex struct {
		specificity int
		index       int
	}
	selected := make(map[string]itemIndex)

	if conditional == nil {
		logging.Error("FilterUnconstrained called with nil conditional")
		rollbar.Error("FilterUnconstrained called with nil conditional")
	}

	for i, item := range items {
		if conditional != nil && item.ConditionalFilter() != "" {
			isTrue, err := conditional.Eval(string(item.ConditionalFilter()))
			if err != nil {
				return nil, err
			}

			if isTrue {
				selected[item.ID()] = itemIndex{0, i}
			}
		}

		if item.ConditionalFilter() == "" {
			constrained, specificity := IsConstrained(item.ConstraintsFilter())
			if !constrained {
				if s, exists := selected[item.ID()]; !exists || s.specificity < specificity {
					selected[item.ID()] = itemIndex{specificity, i}
				}
			}
		}
	}
	indices := make([]int, 0, len(selected))
	for _, s := range selected {
		indices = append(indices, s.index)
	}
	// ensure that the items are returned in the order we get them
	sort.Ints(indices)
	var res []projectfile.ConstrainedEntity
	for _, index := range indices {
		res = append(res, items[index])
	}
	return res, nil
}
