package sysinfo

import "runtime"

// Mainly for testing.
var nameOverride, versionOverride, architectureOverride, libcOverride, compilerOverride string

// Name returns the system's name (e.g. "linux", "windows", etc.)
func Name() string {
	if nameOverride != "" {
		return nameOverride
	}
	return runtime.GOOS
}

// Version returns the system's version.
func Version() string {
	if versionOverride != "" {
		return versionOverride
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
