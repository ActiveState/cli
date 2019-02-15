package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"

	"github.com/ActiveState/cli/internal/environment"

	"github.com/stretchr/testify/assert"
)

func TestCreateUpdate(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "update-generator-test")
	if err != nil {
		log.Fatalf("Cannot create temp dir: %s", err.Error())
	}
	defer os.RemoveAll(dir)

	appPath = os.Args[0]
	version = "1.0"
	genDir = dir

	exitCode := -1
	exit = func(code int) {
		exitCode = code
	}

	os.Chdir(environment.GetRootPathUnsafe())
	run()

	assert.Equal(t, -1, exitCode, "exit was not called")

	assert.FileExists(t, filepath.Join(dir, constants.BranchName, defaultPlatform+".json"), "Should create update bits")
	assert.FileExists(t, filepath.Join(dir, constants.BranchName, "1.0", defaultPlatform+".json"), "Should create update bits")
	assert.FileExists(t, filepath.Join(dir, constants.BranchName, "1.0", defaultPlatform+".gz"), "Should create update bits")
}
