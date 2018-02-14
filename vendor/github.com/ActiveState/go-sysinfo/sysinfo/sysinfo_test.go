package sysinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOSName(t *testing.T) {
	osNameOverride = "linux"
	assert.Equal(t, "linux", OSName())
	osNameOverride = "" // reset
}

func TestOSVersion(t *testing.T) {
	osVersionOverride = "4.0"
	assert.Equal(t, "4.0", OSVersion())
	osVersionOverride = "" // reset
}

func TestOSArchitecture(t *testing.T) {
	osArchitectureOverride = "amd64"
	assert.Equal(t, "amd64", OSArchitecture())
	osArchitectureOverride = "" // reset
}

func TestOSLibc(t *testing.T) {
	osLibcOverride = "glibc-2.25"
	assert.Equal(t, "glibc-2.25", OSLibc())
	osLibcOverride = "" // reset
}

func TestCompiler(t *testing.T) {
	osCompilerOverride = "gcc-7"
	assert.Equal(t, "gcc-7", OSCompiler())
	osCompilerOverride = "" // reset
}
