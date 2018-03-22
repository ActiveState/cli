package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// This package is not fully tested through this test as it is meant for temporary/dev use and the data tested would take
// far too long to mock for the given use-case

func TestGetPackagePaths(t *testing.T) {
	packages := getPackagePathsGo(os.Getenv("GOPATH"))
	assert.NotEqual(t, 0, len(packages), "Should return packages")
}
