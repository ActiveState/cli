package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	target := filepath.Join(environment.GetRootPathUnsafe(), "internal", "constants", "generated.go")
	err := os.Remove(target)
	assert.NoError(t, err, "Removed generated file")

	run()

	assert.FileExists(t, target, "File is generated")
}
