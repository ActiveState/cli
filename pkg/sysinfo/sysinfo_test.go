package sysinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOSVersionInfoCached(t *testing.T) {
	osVersionInfo1, _ := OSVersion()
	osVersionInfo2, _ := OSVersion()
	assert.True(t, osVersionInfo1 != nil, "OSVersion should not be nil")
	assert.True(t, osVersionInfo1 == osVersionInfo2, "Pointers should be equal")
}

func TestLibcInfoCached(t *testing.T) {
	libcInfo1, _ := Libc()
	libcInfo2, _ := Libc()
	assert.True(t, libcInfo1 != nil, "Libc should not be nil")
	assert.True(t, libcInfo1 == libcInfo2, "Pointers should be equal")
}

func TestCompilersCached(t *testing.T) {
	compilers1, _ := Compilers()
	compilers2, _ := Compilers()
	assert.True(t, compilers1 != nil, "Compilers should not be nil")
	for i, _ := range compilers1 {
		assert.True(t, compilers1[i] == compilers2[i], "Pointers should be equal")
	}
}
