package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	inTest = true
}

func TestGenerate(t *testing.T) {
	targetDir, err := ioutil.TempDir("", "constants-generator-test")
	require.NoError(t, err)
	target := filepath.Join(targetDir, "generated.go")
	if _, err := os.Stat(target); err == nil {
		err = os.Remove(target)
		require.NoError(t, err, "Removed generated file")
	}

	run([]string{"", target})
	assert.FileExists(t, target, "File is generated")

	err = os.Remove(target)
	require.NoError(t, err, "Removed generated file")

	run([]string{"", "--", target})
	assert.FileExists(t, target, "File is generated")
}
