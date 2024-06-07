package constraints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/stretchr/testify/assert"
)

var cwd string

func setProjectDir(t *testing.T) {
	var err error
	cwd, err = environment.GetRootPath()
	assert.NoError(t, err, "Should fetch cwd")
	err = os.Chdir(filepath.Join(cwd, "internal", "constraints", "testdata"))
	assert.NoError(t, err, "Should change dir without issue.")
}

// This test is not for constraints, but verifies that sysinfo is working
// correctly in a Linux development environment such that constraints will have
// an effect.
func TestSysinfoLinuxEnv(t *testing.T) {
	if sysinfo.OS() != sysinfo.Linux || os.Getenv("CIRCLECI") != "" {
		return // skip
	}
	assert.Equal(t, sysinfo.Linux, sysinfo.OS(), "Linux is the OS")
	version, err := sysinfo.OSVersion()
	assert.NoError(t, err, "No errors detecting OS version")
	assert.True(t, version.Major > 0, "Determined kernel version")
	assert.NotEqual(t, sysinfo.UnknownArch, sysinfo.Architecture(), "Architecture was recognized")
	libc, err := sysinfo.Libc()
	assert.NoError(t, err, "No errors detecting a Libc")
	assert.NotEqual(t, sysinfo.UnknownLibc, libc.Name, "Libc name was recognized")
	assert.True(t, libc.Major > 0, "Determined Libc version")
	compilers, err := sysinfo.Compilers()
	assert.NoError(t, err, "No errors detecting a compiler")
	for _, compiler := range compilers {
		assert.True(t, compiler.Major > 0, "Determined compiler version")
	}
}

// This test is not for constraints, but verifies that sysinfo is working
// correctly in a Windows development environment such that constraints will
// have an effect.
func TestSysinfoWindowsEnv(t *testing.T) {
	if sysinfo.OS() != sysinfo.Windows || os.Getenv("CIRCLECI") != "" {
		return // skip
	}
	assert.Equal(t, sysinfo.Windows, sysinfo.OS(), "Windows is the OS")
	version, err := sysinfo.OSVersion()
	assert.NoError(t, err, "No errors detecting OS version")
	assert.True(t, version.Major > 0, "Determined OS version")
	assert.NotEqual(t, sysinfo.UnknownArch, sysinfo.Architecture(), "Architecture was recognized")
	libc, err := sysinfo.Libc()
	assert.NoError(t, err, "No errors detecting a Libc")
	assert.NotEqual(t, sysinfo.UnknownLibc, libc.Name, "Libc name was recognized")
	assert.True(t, libc.Major > 0, "Determined Libc version")
	compilers, err := sysinfo.Compilers()
	assert.NoError(t, err, "No errors detecting a compiler")
	for _, compiler := range compilers {
		assert.True(t, compiler.Major > 0, "Determined compiler version")
	}
}

// This test is not for constraints, but verifies that sysinfo is working
// correctly in a macOS development environment such that constraints will have
// an effect.
func TestSysinfoMacOSEnv(t *testing.T) {
	if sysinfo.OS() != sysinfo.Mac || os.Getenv("CIRCLECI") != "" {
		return // skip
	}
	assert.Equal(t, sysinfo.Mac, sysinfo.OS(), "macOS is the OS")
	version, err := sysinfo.OSVersion()
	assert.NoError(t, err, "No errors detecting OS version")
	assert.True(t, version.Major > 0, "Determined OS version")
	assert.NotEqual(t, sysinfo.UnknownArch, sysinfo.Architecture(), "Architecture was recognized")
	libc, err := sysinfo.Libc()
	assert.NoError(t, err, "No errors detecting a Libc")
	assert.NotEqual(t, sysinfo.UnknownLibc, libc.Name, "Libc name was recognized")
	assert.True(t, libc.Major > 0, "Determined Libc version")
	compilers, err := sysinfo.Compilers()
	assert.NoError(t, err, "No errors detecting a compiler")
	for _, compiler := range compilers {
		assert.True(t, compiler.Major > 0, "Determined compiler version")
	}
}

func TestConditional_Eval(t *testing.T) {
	type fields struct {
		params map[string]interface{}
		funcs  template.FuncMap
	}
	tests := []struct {
		name        string
		fields      fields
		conditional string
		want        bool
		wantErr     bool
	}{
		{
			"Basic Conditional",
			fields{
				map[string]interface{}{"value": true},
				map[string]interface{}{},
			},
			".value",
			true,
			false,
		},
		{
			"Basic Negative Conditional",
			fields{
				map[string]interface{}{"value": false},
				map[string]interface{}{},
			},
			".value",
			false,
			false,
		},
		{
			"Multiple Conditionals",
			fields{
				map[string]interface{}{"value1": "v1", "value2": "v2"},
				map[string]interface{}{},
			},
			`or (eq .value1 "v1") (eq .value2 "notv2")`,
			true,
			false,
		},
		{
			"Multiple Conditionals with False",
			fields{
				map[string]interface{}{"value1": "v1", "value2": "v2"},
				map[string]interface{}{},
			},
			`and (eq .value1 "v1") (eq .value2 "notv2")`,
			false,
			false,
		},
		{
			"Custom Functions",
			fields{
				map[string]interface{}{"value1": "foobar"},
				map[string]interface{}{"HasPrefix": strings.HasPrefix},
			},
			`HasPrefix .value1 "foo"`,
			true,
			false,
		},
		{
			"Invalid Conditional",
			fields{
				map[string]interface{}{},
				map[string]interface{}{},
			},
			`I am not a conditional`,
			false,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Conditional{
				params: tt.fields.params,
				funcs:  tt.fields.funcs,
			}
			got, err := c.Eval(tt.conditional)
			if (err != nil) != tt.wantErr {
				t.Errorf("Eval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Eval() got = %v, want %v", got, tt.want)
			}
		})
	}
}
