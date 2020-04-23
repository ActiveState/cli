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
	return Mac
}

var (
	versionRegex = regexp.MustCompile("^(\\d+)\\D(\\d+)(?:\\D(\\d+))?")
)

// OSVersion returns the system's OS version.
func OSVersion() (*OSVersionInfo, error) {
	// Fetch OS version.
	version, err := getDarwinProductVersion()
	if err != nil {
		return nil, fmt.Errorf("Unable to determine OS version: %v", err)
	}

	// Parse OS version parts.
	parts := versionRegex.FindStringSubmatch(version)
	if len(parts) == 0 {
		return nil, fmt.Errorf("Unable to parse version string '%s'", version)
	}

	major, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("Unable to parse part '%s' of version string '%s'", parts[1], version)
	}

	minor, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("Unable to parse part '%s' of version string '%s'", parts[2], version)
	}

	var micro int = 0
	if parts[3] != "" {
		micro, err = strconv.Atoi(parts[3])
		if err != nil {
			return nil, fmt.Errorf("Unable to parse part '%s' of version string '%s'", parts[3], version)
		}
	}
	// Fetch OS name.
	name, err := exec.Command("sw_vers", "-productName").Output()
	return &OSVersionInfo{version, major, minor, micro, string(name)}, nil
}

// Libc returns the system's C library.
func Libc() (*LibcInfo, error) {
	version, err := exec.Command("clang", "--version").Output()
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch libc version: %s", err)
	}
	regex := regexp.MustCompile("(\\d+)\\D(\\d+)")
	parts := regex.FindStringSubmatch(string(version))
	if len(parts) != 3 {
		return nil, fmt.Errorf("Unable to parse libc string '%s'", version)
	}
	for i := 1; i < len(parts); i++ {
		if _, err := strconv.Atoi(parts[i]); err != nil {
			return nil, fmt.Errorf("Unable to parse part '%s' of libc string '%s'", parts[i], version)
		}
	}
	major, _ := strconv.Atoi(parts[1])
	minor, _ := strconv.Atoi(parts[2])
	return &LibcInfo{BsdLibc, major, minor}, nil
}

// Compilers returns the system's available compilers.
func Compilers() ([]*CompilerInfo, error) {
	compilers := []*CompilerInfo{}

	// Map of compiler commands to CompilerNameInfos.
	var compilerMap = map[string]CompilerNameInfo{
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

func getDarwinProductVersion() (string, error) {
	version, err := exec.Command("sw_vers", "-productVersion").Output()
	if err == nil {
		return string(bytes.TrimSpace(version)), nil
	}
	swversErr := err

	plistBuddyArgs := []string{
		"-c",
		"Print:ProductVersion",
		"/System/Library/CoreServices/SystemVersion.plist",
	}
	version, err = exec.Command("/usr/libexec/PlistBuddy", plistBuddyArgs...).Output()
	if err != nil {
		fmt.Sprintf("PlistBuddy error: %v. swver error: %v", err, swversErr)
	}

	return string(bytes.TrimSpace(version)), nil
}
