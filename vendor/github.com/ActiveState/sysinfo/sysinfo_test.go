package sysinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOS(t *testing.T) {
	assert.NotEqual(t, UnknownOs, OS(), "Determined OS")
	assert.NotEqual(t, UnknownOs.String(), OS().String(), "OS is known")
}

func TestOSVersion(t *testing.T) {
	version, err := OSVersion()
	assert.Nil(t, err, "Determined OS version")
	assert.NotEmpty(t, version.Version, "Detected OS version string")
	assert.NotEqual(t, 0, version.Major, "Detected OS version major")
	assert.NotEqual(t, 0, version.Minor, "Detected OS version minor")
	assert.NotEmpty(t, version.Name, "Detected OS name")
}

func TestArchitecture(t *testing.T) {
	assert.NotEqual(t, UnknownArch, Architecture(), "Determined OS architecture")
	assert.NotEqual(t, UnknownArch.String(), Architecture().String(), "Architecture is known")
}

func TestLibc(t *testing.T) {
	libc, err := Libc()
	assert.Nil(t, err, "Determined Libc version")
	assert.NotEqual(t, UnknownLibc, libc.Name, "Determined OS Libc")
	assert.NotEqual(t, UnknownLibc.String(), libc.Name.String(), "Libc is known")
}

func TestCompiler(t *testing.T) {
	compilers, err := Compilers()
	assert.Nil(t, err, "Determined system compilers")
	assert.NotEqual(t, 0, len(compilers), "At least one compiler was found")
	assert.NotEmpty(t, compilers[0].Name.String(), "Compiler has a name")
}
