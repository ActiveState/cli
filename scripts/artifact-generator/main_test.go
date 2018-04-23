package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"

	"github.com/stretchr/testify/assert"
)

// This package is not fully tested through this test as it is meant for temporary/dev use and the data tested would take
// far too long to mock for the given use-case

func TestGetPackagePaths(t *testing.T) {
	packages := getPackagePathsGo(os.Getenv("GOPATH"))
	assert.NotEqual(t, 0, len(packages), "Should return packages")
}

func TestGetRelocatePython(t *testing.T) {
	var path = filepath.Join(environment.GetRootPathUnsafe(), "scripts", "artifact-generator", "source", "vendor", "python3", "distribution", "linux")
	var relocate = getRelocatePython(path, "3.5")
	assert.NotEmpty(t, relocate)

	path = filepath.Join(environment.GetRootPathUnsafe(), "scripts", "artifact-generator", "source", "vendor", "python3", "distribution", "windows")
	relocate = getRelocatePython(path, "3.5")
	assert.NotEmpty(t, relocate)

	path = filepath.Join(environment.GetRootPathUnsafe(), "scripts", "artifact-generator", "source", "vendor", "python3", "distribution", "macos")
	relocate = getRelocatePython(path, "3.5")
	assert.NotEmpty(t, relocate)

	path = filepath.Join(environment.GetRootPathUnsafe(), "scripts", "artifact-generator", "source", "vendor", "python2", "distribution", "linux")
	relocate = getRelocatePython(path, "2.7")
	assert.NotEmpty(t, relocate)

	path = filepath.Join(environment.GetRootPathUnsafe(), "scripts", "artifact-generator", "source", "vendor", "python2", "distribution", "windows")
	relocate = getRelocatePython(path, "2.7")
	assert.NotEmpty(t, relocate)

	path = filepath.Join(environment.GetRootPathUnsafe(), "scripts", "artifact-generator", "source", "vendor", "python2", "distribution", "macos")
	relocate = getRelocatePython(path, "2.7")
	assert.NotEmpty(t, relocate)
}
