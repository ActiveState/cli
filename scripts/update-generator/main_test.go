package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
)

func TestCreateUpdate(t *testing.T) {
	executable := os.Args[0]

	dir, err := ioutil.TempDir(os.TempDir(), "update-generator-test")
	if err != nil {
		log.Fatalf("Cannot create temp dir: %s", err.Error())
	}
	defer os.RemoveAll(dir)

	exitCode := -1
	exit = func(code int) {
		exitCode = code
	}

	os.Chdir(environment.GetRootPathUnsafe())
	os.Args = []string{"", "-o", dir, executable, "1.0"}
	run()

	require.Equal(t, -1, exitCode, "exit was not called")

	_, ext, _ := archiveMeta()

	assert.FileExists(t, filepath.Join(dir, constants.BranchName, defaultPlatform+".json"), "Should create update bits")
	assert.FileExists(t, filepath.Join(dir, constants.BranchName, "1.0", defaultPlatform+".json"), "Should create update bits")
	assert.FileExists(t, filepath.Join(dir, constants.BranchName, "1.0", defaultPlatform+ext), "Should create update bits")

	// Test with branch override
	os.Chdir(environment.GetRootPathUnsafe())
	branchName := "foo"
	os.Args = []string{"", "--b", branchName, "-o", dir, executable, "1.0"}
	run()

	require.Equal(t, -1, exitCode, "exit was not called")

	assert.FileExists(t, filepath.Join(dir, branchName, defaultPlatform+".json"), "Should create update bits")
	assert.FileExists(t, filepath.Join(dir, branchName, "1.0", defaultPlatform+".json"), "Should create update bits")
	assert.FileExists(t, filepath.Join(dir, branchName, "1.0", defaultPlatform+ext), "Should create update bits")
}
