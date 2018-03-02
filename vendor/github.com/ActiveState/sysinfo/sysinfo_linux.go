package sysinfo

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
)

// OS returns the system's OS
func OS() OsInfo {
	return Linux
}

// OSVersion returns the system's OS version.
func OSVersion() (OSVersionInfo, error) {
	// Fetch kernel version.
	version, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return OSVersionInfo{}, fmt.Errorf("Unable to determine OS version: %s", err)
	}
	version = bytes.TrimSpace(version)
	// Parse kernel version parts.
	regex := regexp.MustCompile("^(\\d+)\\D(\\d+)\\D(\\d+)")
	parts := regex.FindStringSubmatch(string(version))
	if len(parts) != 4 {
		return OSVersionInfo{}, fmt.Errorf("Unable to parse version string '%s'", version)
	}
	for i := 1; i < len(parts); i++ {
		if _, err := strconv.Atoi(parts[i]); err != nil {
			return OSVersionInfo{}, fmt.Errorf("Unable to parse part '%s' of version string '%s'", parts[i], version)
		}
	}
	major, _ := strconv.Atoi(parts[1])
	minor, _ := strconv.Atoi(parts[2])
	micro, _ := strconv.Atoi(parts[3])
	// Fetch distribution name.
	// lsb_release -d returns output of the form "Description:\t[Name]".
	name, err := exec.Command("lsb_release", "-d").Output()
	if err == nil && len(bytes.Split(name, []byte(":"))) > 1 {
		name = bytes.TrimSpace(bytes.SplitN(name, []byte(":"), 2)[1])
	} else {
		name = []byte("Unknown")
	}
	return OSVersionInfo{string(version), major, minor, micro, string(name)}, nil
}

// Libc returns the system's C library.
func Libc() (LibcInfo, error) {
	// Assume glibc for now, which exposes a "getconf" command.
	libc, err := exec.Command("getconf", "GNU_LIBC_VERSION").Output()
	if err != nil {
		return LibcInfo{}, fmt.Errorf("Unable to fetch glibc version: %s", err)
	}
	regex := regexp.MustCompile("(\\d+)\\D(\\d+)")
	parts := regex.FindStringSubmatch(string(libc))
	if len(parts) != 3 {
		return LibcInfo{}, fmt.Errorf("Unable to parse libc string '%s'", libc)
	}
	for i := 1; i < len(parts); i++ {
		if _, err := strconv.Atoi(parts[i]); err != nil {
			return LibcInfo{}, fmt.Errorf("Unable to parse part '%s' of libc string '%s'", parts[i], libc)
		}
	}
	major, _ := strconv.Atoi(parts[1])
	minor, _ := strconv.Atoi(parts[2])
	return LibcInfo{Glibc, major, minor}, nil
}

// Map of compiler commands to CompilerNameInfos.
var compilerMap = map[string]CompilerNameInfo{
	"gcc":   Gcc,
	"clang": Clang,
}

// Checks whether or not the given compiler exists and returns its major and
// minor version numbers. A major return of 0 indicates the compiler does not
// exist, or that an error occurred.
func getCompilerVersion(name string) (int, int, error) {
	cc, err := exec.Command(name, "--version").Output()
	if err == nil {
		regex := regexp.MustCompile("(\\d+)\\D(\\d+)\\D\\d+")
		parts := regex.FindStringSubmatch(string(cc))
		if len(parts) != 3 {
			return 0, 0, fmt.Errorf("Unable to parse version string '%s'", cc)
		}
		for i := 1; i < len(parts); i++ {
			if _, err := strconv.Atoi(parts[i]); err != nil {
				return 0, 0, fmt.Errorf("Unable to parse part '%s' of version string '%s'", parts[i], cc)
			}
		}
		major, _ := strconv.Atoi(parts[1])
		minor, _ := strconv.Atoi(parts[2])
		return major, minor, nil
	}
	return 0, 0, nil
}

// Compilers returns the system's available compilers.
func Compilers() ([]CompilerInfo, error) {
	compilers := []CompilerInfo{}

	for command, nameInfo := range compilerMap {
		major, minor, err := getCompilerVersion(command)
		if err != nil {
			return compilers, err
		} else if major > 0 {
			compilers = append(compilers, CompilerInfo{nameInfo, major, minor})
		}
	}

	return compilers, nil
}
