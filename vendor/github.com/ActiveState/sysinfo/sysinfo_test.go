package sysinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	nameOverride = "linux"
	assert.Equal(t, "linux", Name())
	nameOverride = "" // reset
}

func TestVersion(t *testing.T) {
	versionOverride = "4.0"
	assert.Equal(t, "4.0", Version())
	versionOverride = "" // reset
}

func TestArchitecture(t *testing.T) {
	architectureOverride = "amd64"
	assert.Equal(t, "amd64", Architecture())
	architectureOverride = "" // reset
}

func TestLibc(t *testing.T) {
	libcOverride = "glibc-2.25"
	assert.Equal(t, "glibc-2.25", Libc())
	libcOverride = "" // reset
}

func TestCompiler(t *testing.T) {
	compilerOverride = "gcc-7"
	assert.Equal(t, "gcc-7", Compiler())
	compilerOverride = "" // reset
}
