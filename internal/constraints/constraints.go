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

// Map of sysinfo.OSInfos to our constraint OS names.
// Our constraint names may be different from sysinfo names.
var osNames = map[sysinfo.OsInfo]string{
	sysinfo.Linux:   "linux",
	sysinfo.Windows: "windows",
	sysinfo.Mac:     "darwin",
}

// Map of sysinfo.ArchInfos to our constraint arch names.
// Our constraint names may be different from sysinfo names.
var archNames = map[sysinfo.ArchInfo]string{
	sysinfo.I386:  "386",
	sysinfo.Amd64: "amd64",
}

// Map of sysinfo.LibcNameInfos to our constraint libc names.
// Our constraint names may be different from sysinfo names.
var libcNames = map[sysinfo.LibcNameInfo]string{
	sysinfo.Glibc:   "glibc",
	sysinfo.Msvcrt:  "msvcrt",
	sysinfo.BsdLibc: "bsdlibc",
}

// Map of sysinfo.CompilerNameInfos to our constraint compiler names.
// Our constraint names may be different from sysinfo names.
var compilerNames = map[sysinfo.CompilerNameInfo]string{
	sysinfo.Gcc:   "gcc",
	sysinfo.Msvc:  "cl",
	sysinfo.Mingw: "mingw",
	sysinfo.Clang: "clang",
}

// For testing.
var osOverride, osVersionOverride, archOverride, libcOverride, compilerOverride string

// Returns whether or not the sysinfo-detected OS matches the given one
// (presumably the constraint).
func osMatches(os string) bool {
	sysOS := sysinfo.OS()
	if osOverride != "" {
		switch osOverride {
		case osNames[sysinfo.Linux]:
			sysOS = sysinfo.Linux
		case osNames[sysinfo.Windows]:
			sysOS = sysinfo.Windows
		case osNames[sysinfo.Mac]:
			sysOS = sysinfo.Mac
		default:
			sysOS = sysinfo.UnknownOs
		}
	}
	if name, ok := osNames[sysOS]; ok {
		return name == os
	}
	return false
}

// Returns whether or not the sysinfo-detected OS version is greater than or
// equal to the given one (presumably the constraint).
// An example version constraint is "4.1.0".
func osVersionMatches(version string) bool {
	osVersion, err := sysinfo.OSVersion()
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

// Returns whether or not the sysinfo-detected platform architecture matches the
// given one (presumably the constraint).
func archMatches(arch string) bool {
	osArch := sysinfo.Architecture()
	if archOverride != "" {
		switch archOverride {
		case archNames[sysinfo.I386]:
			osArch = sysinfo.I386
		case archNames[sysinfo.Amd64]:
			osArch = sysinfo.Amd64
		default:
			osArch = sysinfo.UnknownArch
		}
	}
	if name, ok := archNames[osArch]; ok {
		return name == arch
	}
	return false
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
		switch name {
		case libcNames[sysinfo.Glibc]:
			osLibc.Name = sysinfo.Glibc
		case libcNames[sysinfo.Msvcrt]:
			osLibc.Name = sysinfo.Msvcrt
		case libcNames[sysinfo.BsdLibc]:
			osLibc.Name = sysinfo.BsdLibc
		default:
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
	if name, ok := libcNames[osLibc.Name]; !ok || name != strings.ToLower(matches[1]) {
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
		switch name {
		case compilerNames[sysinfo.Gcc]:
			osCompilers[0].Name = sysinfo.Gcc
		case compilerNames[sysinfo.Msvc]:
			osCompilers[0].Name = sysinfo.Msvc
		case compilerNames[sysinfo.Mingw]:
			osCompilers[0].Name = sysinfo.Mingw
		case compilerNames[sysinfo.Clang]:
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
		if name, ok := compilerNames[osCompiler.Name]; !ok || name != strings.ToLower(matches[1]) {
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

// Returns whether or not the given platform is constrained by the given
// constraint name.
// If the constraint name is prefixed by "-", returns the converse.
func platformIsConstrainedByConstraintName(platform projectfile.Platform, name string) bool {
	if platform.Name == strings.TrimLeft(name, "-") {
		if PlatformMatches(platform) {
			if strings.HasPrefix(name, "-") {
				return true
			}
		} else if !strings.HasPrefix(name, "-") {
			return true
		}
	}
	return false
}

// Returns whether or not the current platform is constrained by the given
// named constraints, which are defined in the given project configuration.
func platformIsConstrained(constraintNames string) bool {
	project := projectfile.Get()
	for _, name := range strings.Split(constraintNames, ",") {
		for _, platform := range project.Platforms {
			if platformIsConstrainedByConstraintName(platform, name) {
				return true
			}
		}
	}
	return false
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
func IsConstrained(constraint projectfile.Constraint) bool {
	return platformIsConstrained(constraint.Platform) ||
		environmentIsConstrained(constraint.Environment)
}
