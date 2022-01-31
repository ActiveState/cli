package sysinfo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/patrickmn/go-cache"
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
	if cached, found := sysinfoCache.Get(osVersionInfoCacheKey); found {
		return cached.(*OSVersionInfo), nil
	}

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
	info := &OSVersionInfo{version, major, minor, micro, string(name)}
	sysinfoCache.Set(osVersionInfoCacheKey, info, cache.NoExpiration)
	return info, nil
}

// Libc returns the system's C library.
func Libc() (*LibcInfo, error) {
	if cached, found := sysinfoCache.Get(libcInfoCacheKey); found {
		return cached.(*LibcInfo), nil
	}

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
	info := &LibcInfo{Glibc, major, minor}
	sysinfoCache.Set(libcInfoCacheKey, info, cache.NoExpiration)
	return info, nil
}

// Compilers returns the system's available compilers.
func Compilers() ([]*CompilerInfo, error) {
	if cached, found := sysinfoCache.Get(compilersCacheKey); found {
		return cached.([]*CompilerInfo), nil
	}

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

	sysinfoCache.Set(compilersCacheKey, compilers, cache.NoExpiration)
	return compilers, nil
}
