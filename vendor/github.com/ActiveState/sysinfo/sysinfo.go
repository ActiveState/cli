package sysinfo

import "runtime"

// Mainly for testing.
var osOverride, osVersionOverride, architectureOverride, libcOverride, compilerOverride string

// OS returns the system's OS name (e.g. "linux", "windows", etc.)
func OS() string {
	if osOverride != "" {
		return osOverride
	}
	return runtime.GOOS
}

// OSVersion returns the system's OS version.
func OSVersion() string {
	if osVersionOverride != "" {
		return osVersionOverride
	}
	return "" // TODO
}

// Architecture returns the system's architecture (e.g. "amd64", "386", etc.).
func Architecture() string {
	if architectureOverride != "" {
		return architectureOverride
	}
	return runtime.GOARCH
}

// Libc returns the system's libc (e.g. glibc, msvc, etc.) version.
func Libc() string {
	if libcOverride != "" {
		return libcOverride
	}
	return "" // TODO
}

// Compiler returns the system's compiler (e.g. gcc, msvc, etc.) version.
func Compiler() string {
	if compilerOverride != "" {
		return compilerOverride
	}
	return "" // TODO
}
