package sysinfo

import "runtime"

// Mainly for testing.
var osNameOverride, osVersionOverride, osArchitectureOverride, osLibcOverride, osCompilerOverride string

// OSName returns the name of the current OS (e.g. "linux", "windows", etc.)
func OSName() string {
	if osNameOverride != "" {
		return osNameOverride
	}
	return runtime.GOOS
}

// OSVersion returns the current OS version.
func OSVersion() string {
	if osVersionOverride != "" {
		return osVersionOverride
	}
	return "" // TODO
}

// OSArchitecture returns the current OS architecture (e.g. "amd64", "386",
// etc.).
func OSArchitecture() string {
	if osArchitectureOverride != "" {
		return osArchitectureOverride
	}
	return runtime.GOARCH
}

// OSLibc returns the current OS's libc (e.g. glibc, msvc, etc.) version.
func OSLibc() string {
	if osLibcOverride != "" {
		return osLibcOverride
	}
	return "" // TODO
}

// OSCompiler returns the current OS's compiler (e.g. gcc, msvc, etc.) version.
func OSCompiler() string {
	if osCompilerOverride != "" {
		return osCompilerOverride
	}
	return "" // TODO
}
