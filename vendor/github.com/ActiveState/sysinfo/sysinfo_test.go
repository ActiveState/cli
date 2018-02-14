package sysinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOS(t *testing.T) {
	osOverride = "linux"
	assert.Equal(t, "linux", OS())
	osOverride = "" // reset
}

func TestOSVersion(t *testing.T) {
	osVersionOverride = "4.0"
	assert.Equal(t, "4.0", OSVersion())
	osVersionOverride = "" // reset
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
