package constraints

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"text/template"

	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var cwd string

func setProjectDir(t *testing.T) {
	var err error
	cwd, err = environment.GetRootPath()
	assert.NoError(t, err, "Should fetch cwd")
	err = os.Chdir(filepath.Join(cwd, "internal", "constraints", "testdata"))
	assert.NoError(t, err, "Should change dir without issue.")
	projectfile.Reset()
}

func TestOsConstraints(t *testing.T) {
	osname := sysinfo.OS().String()
	exclude := "-" + osname

	//Test single
	if sysinfo.OS() == sysinfo.Windows {
		assert.False(t, osIsConstrained(osname))
		assert.True(t, osIsConstrained(exclude))
		assert.True(t, osIsConstrained("macos"))
		assert.True(t, osIsConstrained("Linux"))
	}
	if sysinfo.OS() == sysinfo.Mac {
		assert.False(t, osIsConstrained(osname))
		assert.True(t, osIsConstrained(exclude))
		assert.True(t, osIsConstrained("linux"))
		assert.True(t, osIsConstrained("windows"))
	}
	if sysinfo.OS() == sysinfo.Linux {
		assert.False(t, osIsConstrained(osname))
		assert.True(t, osIsConstrained(exclude))
		assert.True(t, osIsConstrained("macos"))
		assert.True(t, osIsConstrained("windows"))
	}
	// Test multiple
	assert.False(t, osIsConstrained("linux,windows,macos"))
	assert.True(t, osIsConstrained(fmt.Sprintf("linux,windows,macos,%s", exclude)))
}

func TestPlatformConstraints(t *testing.T) {
	setProjectDir(t)
	exclude := "-linux-label"
	if sysinfo.OS() == sysinfo.Windows {
		exclude = "-windows-label"
	} else if sysinfo.OS() == sysinfo.Mac {
		exclude = "-macos-label"
	}
	if sysinfo.OS() != sysinfo.Windows {
		assert.True(t, platformIsConstrained("Windows10Label"))
	}
	assert.False(t, platformIsConstrained("windows-label,linux-label,macos-label"), "No matter the platform, this should never be constrained.")
	assert.True(t, platformIsConstrained(fmt.Sprintf("windows-label,linux-label,macos-label,%s", exclude)), "Exclude at the end is still considered.")
	assert.True(t, platformIsConstrained(fmt.Sprintf("%s,windows-label,linux-label,macos-label", exclude)), "Exclude at the start means (or any part really) means fail.")
}

func TestEnvironmentConstraints(t *testing.T) {
	os.Setenv(constants.EnvironmentEnvVarName, "dev")
	assert.False(t, environmentIsConstrained("dev"), "The current environment is in 'dev'")
	assert.False(t, environmentIsConstrained("dev,qa"), "The current environment is in 'dev,qa'")
	assert.False(t, environmentIsConstrained("qa,dev,prod"), "The current environment is in 'dev,qa,prod'")
	assert.True(t, environmentIsConstrained("qa"), "The current environment is not in 'qa'")
	assert.True(t, environmentIsConstrained("qa,devops"), "The current environment is not in 'qa,devops'")
}

func TestMatchConstraint(t *testing.T) {
	root, _ := environment.GetRootPath()
	project, err := projectfile.Parse(filepath.Join(root, "test", constants.ConfigFileName))
	project.Persist()
	assert.Nil(t, err, "There was no error parsing the config file")

	variableLabel := "Windows10Label"
	if sysinfo.OS() == sysinfo.Windows {
		variableLabel = "Linux64Label"
	}

	constraint := projectfile.Constraint{sysinfo.OS().String(), variableLabel, "dev"}
	constrained, specificity := IsConstrained(constraint)
	assert.True(t, constrained)
	assert.Equal(t, 3, specificity)
	beConstrained := "windows"
	if sysinfo.OS() == sysinfo.Windows {
		beConstrained = "linux"
	}
	constrained, specificity = IsConstrained(projectfile.Constraint{beConstrained, "", ""})
	assert.True(t, constrained)
	assert.Equal(t, 1, specificity)

	constrained, specificity = IsConstrained(projectfile.Constraint{sysinfo.OS().String(), "", ""})
	// Confirm passing only one constraint doesn't get constrained when we don't expect
	assert.False(t, constrained)
	assert.Equal(t, 1, specificity)
	{
		osOverride = "windows"
		osVersionOverride = "10"
		constrained, specificity = IsConstrained(projectfile.Constraint{"", "Windows10Label", ""})
		assert.False(t, constrained)
		assert.Equal(t, 1, specificity)
		osOverride = ""
		osVersionOverride = ""
	}
	{
		os.Setenv("ACTIVESTATE_ENVIRONMENT", "itworks")
		constrained, specificity = IsConstrained(projectfile.Constraint{"", "", "itworks"})
		assert.False(t, constrained)
		assert.Equal(t, 1, specificity)
		os.Setenv("ACTIVESTATE_ENVIRONMENT", "")
	}
	constrained, specificity = IsConstrained(projectfile.Constraint{"", "", ""})
	assert.False(t, constrained)
	assert.Equal(t, 0, specificity)

	// Confirm we DO get constrained with only one value set
	constrained, specificity = IsConstrained(projectfile.Constraint{beConstrained, "", ""})
	assert.True(t, constrained)
	assert.Equal(t, 1, specificity)
	constrained, specificity = IsConstrained(projectfile.Constraint{"", variableLabel, ""})
	assert.True(t, constrained)
	assert.Equal(t, 1, specificity)
	constrained, specificity = IsConstrained(projectfile.Constraint{"", "", "dev"})
	assert.True(t, constrained)
	assert.Equal(t, 1, specificity)

	// Don't constrain at all if nothing is passed in
	constrained, specificity = IsConstrained(projectfile.Constraint{"", "", ""})
	assert.False(t, constrained)
	assert.Equal(t, 0, specificity)
}

func TestOsMatches(t *testing.T) {
	osNames := []string{"linux", "windows", "macos", "Linux", "Windows", "MacOS", "macOS"}
	for _, name := range osNames {
		osOverride = name
		assert.True(t, osMatches(name), "OS matches with override")
	}
	osOverride = "" // reset
}

func TestOsVersionMatches(t *testing.T) {
	// Linux tests.
	osVersionOverride = "4.10.0 Ubuntu 16.04.3 LTS"
	assert.False(t, osVersionMatches("4.10.1"), "Newer kernel required")
	assert.False(t, osVersionMatches("4.11"), "Newer kernel required")
	assert.False(t, osVersionMatches("5"), "Newer kernel required")
	assert.True(t, osVersionMatches("4.10.0"), "Kernel matches")
	assert.True(t, osVersionMatches("4.10"), "Kernel matches")
	assert.True(t, osVersionMatches("4.09.1"), "Older kernel is okay")
	assert.True(t, osVersionMatches("4.09"), "Older kernel is okay")
	assert.True(t, osVersionMatches("4"), "Older kernel is okay")

	// Windows tests.
	osVersionOverride = "6.1.999 Windows 7"
	assert.False(t, osVersionMatches("6.2.0"), "Windows 8 required")
	assert.False(t, osVersionMatches("6.2"), "Windows 8 required")
	assert.False(t, osVersionMatches("10"), "Windows 10 required")
	assert.True(t, osVersionMatches("6.1.0"), "Windows 7 is okay")
	assert.True(t, osVersionMatches("6.0"), "Windows Vista is okay")

	// macOS tests.
	osVersionOverride = "10.6.2 Mac OS X"
	assert.False(t, osVersionMatches("10.7.0"), "Lion required")
	assert.False(t, osVersionMatches("10.7"), "Lion required")
	assert.False(t, osVersionMatches("10.10"), "Mavericks required")
	assert.True(t, osVersionMatches("10.5.0"), "Leopard is okay")
	assert.True(t, osVersionMatches("10.4"), "Tiger is okay")

	osVersionOverride = "" // reset
}

func TestMatchesGlobbed(t *testing.T) {
	assert.True(t, matchesGlobbed("This might test it out", "*test*it*"))
	assert.True(t, matchesGlobbed("testit", "test*it"))
	assert.True(t, matchesGlobbed("test it", "test*it"))
	assert.False(t, matchesGlobbed("This might zest it out", "*test*it*"))
	assert.True(t, matchesGlobbed("zThis it test it out", "*test*it*"))
	assert.False(t, matchesGlobbed("This test it out", "test*it"))
	assert.True(t, matchesGlobbed("test out it", "test*it"))
	assert.True(t, matchesGlobbed("test out it test", "test*it*"))
	assert.False(t, matchesGlobbed("test out it test", "test*it"))
	assert.True(t, matchesGlobbed("test it out it test", "*it*test*"))
}

func TestArchMatches(t *testing.T) {
	archNames := []string{"i386", "x86_64", "arm", "I386", "X86_64", "ARM"}
	for _, name := range archNames {
		archOverride = name
		assert.True(t, archMatches(name), "Architecture matches with override")
	}
	archOverride = "" // reset
}

func TestLibcMatches(t *testing.T) {
	// Linux tests.
	libcOverride = "glibc 2.23"
	assert.False(t, libcMatches("glibc 2.24"), "Newer glibc required")
	assert.False(t, libcMatches("glibc 3.0"), "Newer glibc required")
	assert.True(t, libcMatches("glibc 2.23"), "glibc matches")
	assert.True(t, libcMatches("glibc 2.22"), "Older glibc is okay")
	assert.True(t, libcMatches("glibc 1.0"), "Older glibc is okay")
	assert.False(t, libcMatches("musl 2.23"), "Non-glibc (musl) is not okay")
	assert.False(t, libcMatches("musl 2"), "Non-glibc (musl) is not okay")
	assert.True(t, libcMatches("GLIBC 2.23"), "Case-insensitive matching")

	// Windows tests.
	libcOverride = "msvcrt 7.0"
	assert.False(t, libcMatches("msvcrt 8.0"), "Newer msvcrt required")
	assert.True(t, libcMatches("msvcrt 7.0"), "msvcrt matches")
	assert.True(t, libcMatches("msvcrt 6.0"), "Older msvcrt is okay")
	assert.False(t, libcMatches("glibc 2.23"), "Non-msvcrt (glibc) is not okay")
	assert.True(t, libcMatches("MSVCRT 7.0"), "Case-insensitive matching")

	// macOS tests.
	libcOverride = "libc 3.2"
	assert.False(t, libcMatches("libc 3.4"), "Newer libc required")
	assert.False(t, libcMatches("libc 4.0"), "Newer libc required")
	assert.True(t, libcMatches("libc 3.2"), "libc matches")
	assert.True(t, libcMatches("libc 3.0"), "Older libc is okay")
	assert.True(t, libcMatches("libc 2.0"), "Older libc is okay")
	assert.True(t, libcMatches("LIBC 3.2"), "Case-insensitive matching")

	libcOverride = "" // reset
}

func TestCompilerMatches(t *testing.T) {
	// Linux tests.
	compilerOverride = "gcc 5.2"
	assert.False(t, compilerMatches("gcc 5.4"), "Newer GCC required")
	assert.False(t, compilerMatches("gcc 6"), "Newer GCC required")
	assert.True(t, compilerMatches("gcc 5.2"), "GCC matches")
	assert.True(t, compilerMatches("gcc 5"), "Older GCC is okay")
	assert.True(t, compilerMatches("gcc 4"), "Older GCC is okay")
	assert.False(t, compilerMatches("clang 3.4"), "Non-GCC (Clang) is not okay")
	assert.True(t, compilerMatches("GCC 5.2"), "Case-insensitive matching")

	// Windows tests.
	compilerOverride = "msvc 17.00"
	assert.False(t, compilerMatches("msvc 19.00"), "Newer msvc required")
	assert.False(t, compilerMatches("msvc 19"), "Newer msvc required")
	assert.True(t, compilerMatches("msvc 17.00"), "msvc matches")
	assert.True(t, compilerMatches("msvc 17"), "msvc matches")
	assert.True(t, compilerMatches("msvc 15.00"), "Older msvc is okay")
	assert.True(t, compilerMatches("msvc 15"), "Older msvc is okay")
	assert.False(t, compilerMatches("mingw 5.4"), "Non-msvc (MinGW) is not okay")
	assert.True(t, compilerMatches("MSVC 17"), "Case-insensitive matching")

	// macOS tests.
	compilerOverride = "clang 6.0"
	assert.False(t, compilerMatches("clang 7.0"), "Newer clang required")
	assert.False(t, compilerMatches("clang 7"), "Newer clang required")
	assert.True(t, compilerMatches("clang 6.0"), "clang matches")
	assert.True(t, compilerMatches("clang 6"), "clang matches")
	assert.True(t, compilerMatches("clang 4"), "Older clang is okay")
	assert.True(t, compilerMatches("clang 3.4"), "Older clang is okay")
	assert.True(t, compilerMatches("CLANG 6"), "Case-insensitive matching")

	compilerOverride = "" // reset
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

func sliceContains(s []int, v int) bool {
	for _, sv := range s {
		if sv == v {
			return true
		}
	}
	return false
}

func mockConstraint(unconstrained bool, env ...string) projectfile.Constraint {
	environ := ""
	if len(env) == 1 {
		environ = env[0]
	}
	if unconstrained {
		return projectfile.Constraint{sysinfo.OS().String(), "", environ}
	}

	beConstrained := "windows"
	if sysinfo.OS() == sysinfo.Windows {
		beConstrained = "linux"
	}
	return projectfile.Constraint{beConstrained, "", ""}
}

func TestFilterUnconstrained(t *testing.T) {
	os.Setenv("ACTIVESTATE_ENVIRONMENT", "TEST_ENV")
	defer os.Unsetenv("ACTIVESTATE_ENVIRONMENT")
	cases := []struct {
		Name     string
		Selected []int
		Override bool
	}{
		{"all selected", []int{0, 1, 2}, false},
		{"none selected", []int{}, false},
		{"one selected", []int{1}, false},
		{"one overridden", []int{3, 1, 2}, true},
	}

	for _, c := range cases {

		t.Run(c.Name, func(tt *testing.T) {
			items := make(projectfile.Constants, 0, 4)
			for i := 0; i < 3; i++ {
				items = append(items, &projectfile.Constant{
					Name:        fmt.Sprintf("event%d", i),
					Constraints: mockConstraint(sliceContains(c.Selected, i) || c.Override),
				})
			}
			if c.Override {
				items = append(items, &projectfile.Constant{
					Name:        "event0",
					Constraints: mockConstraint(true, "TEST_ENV"),
				})
			}

			constrained, err := FilterUnconstrained(nil, items.AsConstrainedEntities())
			require.NoError(tt, err)

			res := projectfile.MakeConstantsFromConstrainedEntities(constrained)
			expected := make([]*projectfile.Constant, 0, len(c.Selected))
			for _, ii := range c.Selected {
				expected = append(expected, items[ii])
			}
			sort.Slice(res, func(i, j int) bool {
				return res[i].Name < res[j].Name
			})
			assert.Len(tt, res, len(c.Selected), "select %d unconstrained items", len(c.Selected))
			assert.Equal(tt, expected, res, "select unconstrained items")
		})
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