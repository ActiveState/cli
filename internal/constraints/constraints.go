package constraints

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/sysinfo"
)

// For testing.
var osOverride, osVersionOverride, archOverride, libcOverride, compilerOverride string

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
	osVersion, err := sysinfo.OSVersion()

	if osVersionMatchesGlobbed(osVersion.Version, version) {
		return true
	}

	if osVersionOverride != "" {
		// When writing tests, this string should be of the form:
		// [major].[minor].[micro] [os free-form name]
		osVersion = &sysinfo.OSVersionInfo{}
		fmt.Sscanf(osVersionOverride, "%d.%d.%d %s", &osVersion.Major, &osVersion.Minor, &osVersion.Micro, &osVersion.Name)
		osVersion.Version = fmt.Sprintf("%d.%d.%d", osVersion.Major, osVersion.Minor, osVersion.Micro)
		err = nil
	}
	if err != nil {
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
	osLibc, err := sysinfo.Libc()
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
		err = nil
	}
	if err != nil {
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
	osCompilers, err := sysinfo.Compilers()
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
		err = nil
	}
	if err != nil {
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
func FilterUnconstrained(items []projectfile.ConstrainedEntity) []int {
	type itemIndex struct {
		specificity int
		index       int
	}
	selected := make(map[string]itemIndex)

	for i, item := range items {
		c := item.ConstraintsFilter()
		constrained, specificity := IsConstrained(c)
		if !constrained {
			if s, exists := selected[item.ID()]; !exists || s.specificity < specificity {
				selected[item.ID()] = itemIndex{specificity, i}
			}
		}
	}
	res := make([]int, 0, len(selected))
	for _, s := range selected {
		res = append(res, s.index)
	}
	return res
}

// FilterUnconstrainedEvents filters events that are not constrained.
// If two events with the same name exist are unconstrained, only the most
// specific one will be returned.
func FilterUnconstrainedEvents(events []projectfile.Event) []*projectfile.Event {
	items := make([]projectfile.ConstrainedEntity, 0, len(events))
	for _, ev := range events {
		items = append(items, ev)
	}
	filtered := FilterUnconstrained(items)
	res := make([]*projectfile.Event, 0, len(filtered))
	for _, item := range filtered {
		res = append(res, &events[item])
	}
	return res
}

// FilterUnconstrainedLanguages filters languages that are not constrained.
// If two languages with the same name exist are unconstrained, only the most
// specific one will be returned.
func FilterUnconstrainedLanguages(languages []projectfile.Language) []*projectfile.Language {
	items := make([]projectfile.ConstrainedEntity, 0, len(languages))
	for _, l := range languages {
		items = append(items, l)
	}
	filtered := FilterUnconstrained(items)
	res := make([]*projectfile.Language, 0, len(filtered))
	for _, item := range filtered {
		res = append(res, &languages[item])
	}
	return res
}

// FilterUnconstrainedScripts filters scripts that are not constrained.
// If two scripts with the same name exist are unconstrained, only the most
// specific one will be returned.
func FilterUnconstrainedScripts(scripts []projectfile.Script) []*projectfile.Script {
	items := make([]projectfile.ConstrainedEntity, 0, len(scripts))
	for _, s := range scripts {
		items = append(items, s)
	}
	filtered := FilterUnconstrained(items)
	res := make([]*projectfile.Script, 0, len(filtered))
	for _, item := range filtered {
		res = append(res, &scripts[item])
	}
	return res
}

// FilterUnconstrainedConstants filters constants that are not constrained.
// If two constants with the same name exist are unconstrained, only the most
// specific one will be returned.
func FilterUnconstrainedConstants(constants []*projectfile.Constant) []*projectfile.Constant {
	items := make([]projectfile.ConstrainedEntity, 0, len(constants))
	for _, c := range constants {
		items = append(items, c)
	}
	filtered := FilterUnconstrained(items)
	res := make([]*projectfile.Constant, 0, len(filtered))
	for _, item := range filtered {
		res = append(res, constants[item])
	}
	return res
}

// FilterUnconstrainedPackages filters packages that are not constrained.
// If two packages with the same name exist are unconstrained, only the most
// specific one will be returned.
func FilterUnconstrainedPackages(packages []projectfile.Package) []*projectfile.Package {
	items := make([]projectfile.ConstrainedEntity, 0, len(packages))
	for _, p := range packages {
		items = append(items, p)
	}
	filtered := FilterUnconstrained(items)
	res := make([]*projectfile.Package, 0, len(filtered))
	for _, item := range filtered {
		res = append(res, &packages[item])
	}
	return res
}

// FilterUnconstrainedSecrets filters secrets that are not constrained.
// If two secrets with the same name exist are unconstrained, only the most
// specific one will be returned.
func FilterUnconstrainedSecrets(secrets []*projectfile.Secret) []*projectfile.Secret {
	items := make([]projectfile.ConstrainedEntity, 0, len(secrets))
	for _, s := range secrets {
		items = append(items, s)
	}
	filtered := FilterUnconstrained(items)
	res := make([]*projectfile.Secret, 0, len(filtered))
	for _, item := range filtered {
		res = append(res, secrets[item])
	}
	return res
}

// MostSpecificUnconstrained searches for entities named name and returns the
// unconstrained with the most specific constraint definition (if it exists).
// It also returns the index of the found item in the list (which is -1 if none
// could be found)
func MostSpecificUnconstrained(name string, items []projectfile.ConstrainedEntity) (int, projectfile.ConstrainedEntity) {
	var maxSpecificity int = -1
	var value projectfile.ConstrainedEntity
	var index int = -1

	for i, item := range items {
		c := item.ConstraintsFilter()
		constrained, specificity := IsConstrained(c)
		if item.ID() == name && !constrained {
			if specificity > maxSpecificity {
				maxSpecificity = specificity
				value = item
				index = i
			}
		}
	}
	return index, value
}

// MostSpecificUnconstrainedEvent searches for events named name and returns the
// unconstrained with the most specific constraint definition (if it exists).
// It also returns the index of the found item in the list (which is -1 if none
// could be found)
func MostSpecificUnconstrainedEvent(name string, events []projectfile.Event) (int, projectfile.Event) {
	items := make([]projectfile.ConstrainedEntity, 0, len(events))
	for _, ev := range events {
		items = append(items, ev)
	}
	i, v := MostSpecificUnconstrained(name, items)
	return i, v.(projectfile.Event)
}

// MostSpecificUnconstrainedConstant searches for constants named name and returns the
// unconstrained with the most specific constraint definition (if it exists).
// It also returns the index of the found item in the list (which is -1 if none
// could be found)
func MostSpecificUnconstrainedConstant(name string, constants []*projectfile.Constant) (int, *projectfile.Constant) {
	items := make([]projectfile.ConstrainedEntity, 0, len(constants))
	for _, c := range constants {
		items = append(items, c)
	}
	i, v := MostSpecificUnconstrained(name, items)
	return i, v.(*projectfile.Constant)
}

// MostSpecificUnconstrainedScript searches for scripts named name and returns the
// unconstrained with the most specific constraint definition (if it exists).
// It also returns the index of the found item in the list (which is -1 if none
// could be found)
func MostSpecificUnconstrainedScript(name string, scripts []projectfile.Script) (int, projectfile.Script) {
	items := make([]projectfile.ConstrainedEntity, 0, len(scripts))
	for _, s := range scripts {
		items = append(items, s)
	}
	i, v := MostSpecificUnconstrained(name, items)
	return i, v.(projectfile.Script)
}
