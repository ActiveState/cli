package sysinfo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strconv"
)

// OS returns the system's OS
func OS() OsInfo {
	return Linux
}

var (
	versionRegex = regexp.MustCompile(`^(\d+)\D(\d+)\D(\d+)`)
)

// OSVersion returns the system's OS version.
func OSVersion() (*OSVersionInfo, error) {
	// Fetch kernel version.
	osrelFile := "/proc/sys/kernel/osrelease"
	osrelData, err := ioutil.ReadFile(osrelFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to read %s: %v", osrelFile, err)
	}
	version := string(bytes.TrimSpace(osrelData))

	// Parse kernel version parts.
	versionParts := versionRegex.FindStringSubmatch(version)
	if len(versionParts) != 4 {
		return nil, fmt.Errorf("Unable to parse version string %q", versionParts)
	}
	major, _ := strconv.Atoi(versionParts[1])
	minor, _ := strconv.Atoi(versionParts[2])
	micro, _ := strconv.Atoi(versionParts[3])
	// Fetch distribution name.
	// lsb_release -d returns output of the form "Description:\t[Name]".
	name, err := exec.Command("lsb_release", "-d").Output()
	if err == nil && len(bytes.Split(name, []byte(":"))) > 1 {
		name = bytes.TrimSpace(bytes.SplitN(name, []byte(":"), 2)[1])
	} else {
		etcFiles := []string{
			"/etc/debian_version", // Debians
			"/etc/redhat-release", // RHELs and Fedoras
			"/etc/system-release", // Amazon AMIs
			"/etc/SuSE-release",   // SuSEs
		}
		for _, etcFile := range etcFiles {
			name, err = ioutil.ReadFile(etcFile)
			if err == nil {
				break
			}
		}
		if bytes.Equal(name, []byte("")) {
			name = []byte("Unknown")
		}
	}
	return &OSVersionInfo{version, major, minor, micro, string(name)}, nil
}

// Libc returns the system's C library.
func Libc() (*LibcInfo, error) {
	// Assume glibc for now, which exposes a "getconf" command.
	libc, err := exec.Command("getconf", "GNU_LIBC_VERSION").Output()
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch glibc version: %s", err)
	}
	regex := regexp.MustCompile("(\\d+)\\D(\\d+)")
	parts := regex.FindStringSubmatch(string(libc))
	if len(parts) != 3 {
		return nil, fmt.Errorf("Unable to parse libc string '%s'", libc)
	}
	major, _ := strconv.Atoi(parts[1])
	minor, _ := strconv.Atoi(parts[2])
	return &LibcInfo{Glibc, major, minor}, nil
}

// Compilers returns the system's available compilers.
func Compilers() ([]*CompilerInfo, error) {
	compilers := []*CompilerInfo{}

	// Map of compiler commands to CompilerNameInfos.
	var compilerMap = map[string]CompilerNameInfo{
		"gcc":   Gcc,
		"clang": Clang,
	}
	for command, nameInfo := range compilerMap {
		major, minor, err := getCompilerVersion([]string{command, "--version"})
		if err != nil {
			return compilers, err
		} else if major > 0 {
			compilers = append(compilers, &CompilerInfo{nameInfo, major, minor})
		}
	}

	return compilers, nil
}
