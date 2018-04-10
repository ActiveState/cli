package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSha256(t *testing.T) {
	sha := generateSha256("foo")
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", sha, "Should generate our SHA256")
}

func TestCreateUpdate(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "update-generator-test")
	if err != nil {
		log.Fatalf("Cannot create temp dir: %s", err.Error())
	}
	defer os.RemoveAll(dir)

	appPath = "build/state"
	version = "1.0"
	genDir = dir

	os.Chdir(environment.GetRootPathUnsafe())
	run()

	assert.FileExists(t, filepath.Join(dir, defaultPlatform+".json"), "Should create update bits")
	assert.FileExists(t, filepath.Join(dir, "1.0", defaultPlatform+".gz"), "Should create update bits")
}
