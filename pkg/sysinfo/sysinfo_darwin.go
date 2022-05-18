package sysinfo

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/patrickmn/go-cache"
)

// OS returns the system's OS
func OS() OsInfo {
	return Mac
}

var (
	plistVersionRegex = regexp.MustCompile("(?s)ProductVersion.*?([\\d\\.]+)")
)

// OSVersion returns the system's OS version.
func OSVersion() (*OSVersionInfo, error) {
	if cached, found := sysinfoCache.Get(osVersionInfoCacheKey); found {
		return cached.(*OSVersionInfo), nil
	}

	if v := os.Getenv(VersionOverrideEnvVar); v != "" {
		vInfo, err := parseVersionInfo(v)
		if err != nil {
			return nil, fmt.Errorf("Could not parse version info: %w", err)
		}
		return &OSVersionInfo{vInfo, "spoofed"}, nil
	}

	// Fetch OS version.
	version, err := getDarwinProductVersion()
	if err != nil {
		return nil, fmt.Errorf("Unable to determine OS version: %v", err)
	}

	vInfo, err := parseVersionInfo(version)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse OS version: %w", err)
	}

	// Fetch OS name.
	name, err := exec.Command("sw_vers", "-productName").Output()
	info := &OSVersionInfo{vInfo, string(name)}
	sysinfoCache.Set(osVersionInfoCacheKey, info, cache.NoExpiration)
	return info, nil
}

// Libc returns the system's C library.
func Libc() (*LibcInfo, error) {
	if cached, found := sysinfoCache.Get(libcInfoCacheKey); found {
		return cached.(*LibcInfo), nil
	}

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
	info := &LibcInfo{BsdLibc, major, minor}
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

func getDarwinProductVersion() (string, error) {
	v, err := getDarwinProductVersionFromFS()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		logging.Warning("Unable to fetch OS version from filesystem: %v", errs.JoinMessage(err))
	} else if err == nil {
		return v, nil
	}

	version, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return "", locale.WrapError(err, "Could not detect your OS version, error received: %s", err.Error())
	}
	return string(bytes.TrimSpace(version)), nil
}

func getDarwinProductVersionFromFS() (string, error) {
	fpath := "/System/Library/CoreServices/SystemVersion.plist"
	if !fileutils.TargetExists(fpath) {
		return "", fs.ErrNotExist
	}

	b, err := fileutils.ReadFile(fpath)
	if err != nil {
		return "", errs.Wrap(err, "Could not read %s", fpath)
	}

	match := plistVersionRegex.FindSubmatch(b)
	if len(match) != 2 {
		return "", errs.Wrap(err, "SystemVersion.plist does not contain a valid ProductVersion, match result: %v, xml:\n%s", match, string(b))
	}

	return string(match[1]), nil
}
