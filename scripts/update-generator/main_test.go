package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/fileutils"
)

func TestCreateUpdate(t *testing.T) {
	latestFlag = p.BoolP(true)
	dir, err := ioutil.TempDir(os.TempDir(), "update-generator-test")
	if err != nil {
		log.Fatalf("Cannot create temp dir: %s", err.Error())
	}

	systemInstall := filepath.Join(dir, "system-install")
	err = fileutils.Mkdir(systemInstall)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	binary1 := filepath.Join(dir, "binary1")
	binary2 := filepath.Join(dir, "binary2")

	for _, f := range []string{binary1, binary2} {
		err = fileutils.Touch(f)
		require.NoError(t, err)
	}

	err = createUpdate(dir, "channel", "version", "platform", dir)
	require.NoError(t, err)

	_, ext := archiveMeta()

	assert.FileExists(t, filepath.Join(dir, "channel", "platform", "info.json"), "Should create update bits")
	assert.FileExists(t, filepath.Join(dir, "channel", "version", "platform", "info.json"), "Should create update bits")
	assert.FileExists(t, filepath.Join(dir, "channel", "version", "platform", "state-platform-version"+ext), "Should create update bits")
}
