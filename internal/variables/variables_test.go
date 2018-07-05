package variables

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func loadProject(t *testing.T) *projectfile.Project {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
platforms:
  - name: Linux
    os: linux
  - name: Windows
    os: windows
  - name: macOS
    os: macos
variables:
  - name: foo
    value: bar
  - name: bar
    value: baz
    constraints:
      platform: Linux
  - name: bar
    value: quux
    constraints:
    platform: Windows
  - name: UPPERCASE
    value: foo
hooks:
  - name: pre
    value: echo 'Hello $variables.foo!'
  - name: post
    value: echo 'Hello $variables.bar!'
commands:
  - name: test
    value: make test
  - name: recursive
    value: $commands.recursive
  `)

	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	return project
}

func TestExpandProjectPlatformOs(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$platform.os", project)
	if runtime.GOOS != "darwin" {
		assert.Equal(t, runtime.GOOS, expanded, "Expanded platform variable")
	} else {
		assert.Equal(t, "macos", expanded, "Expanded platform variable")
	}
}

func TestExpandProjectHook(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$hooks.pre", project)
	assert.Equal(t, "echo 'Hello bar!'", expanded, "Expanded simple variable")
}

func TestExpandProjectHookWithConstraints(t *testing.T) {
	project := loadProject(t)

	if runtime.GOOS == "linux" {
		expanded := ExpandFromProject("$hooks.post", project)
		assert.Equal(t, "echo 'Hello baz!'", expanded, "Expanded platform-specific variable")
	} else if runtime.GOOS == "windows" {
		expanded := ExpandFromProject("$hooks.post", project)
		assert.Equal(t, "echo 'Hello quux!'", expanded, "Expanded platform-specific variable")
	}
}

func TestExpandProjectCommand(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$ $commands.test", project)
	assert.Equal(t, "$ make test", expanded, "Expanded simple command")
}

func TestExpandProjectAlternateSyntax(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("${platform.os}", project)
	if runtime.GOOS != "darwin" {
		assert.Equal(t, runtime.GOOS, expanded, "Expanded platform variable")
	} else {
		assert.Equal(t, "macos", expanded, "Expanded platform variable")
	}
}

func TestExpandProjectUnknownCategory(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$unknown.unknown", project)
	assert.Equal(t, "", expanded, "Failed to expand")
}

func TestExpandProjectUnknownName(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$platform.unknown", project)
	assert.Equal(t, "", expanded, "Failed to expand")
}

func TestExpandProjectInfiniteRecursion(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$commands.recursive", project)
	assert.Equal(t, "", expanded, "Failed to expand")
}

// Tests all possible $platform.[name] variable expansions.
func TestExpandProjectPlatform(t *testing.T) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
  platforms:
    - name: Any
      os: any
      version: any
      architecture: any
      libc: any
      compiler: any
  `)

	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	for _, name := range []string{"name", "os", "version", "architecture", "libc", "compiler"} {
		expanded := ExpandFromProject(fmt.Sprintf("$platform.%s", name), project)
		assert.NotEqual(t, "", expanded, "Expanded without failure: "+name)
	}
}

func TestExpandProjectEmbedded(t *testing.T) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
  variables:
    - name: foo
      value: bar
  `)

	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	expanded := ExpandFromProject("$variables.foo is in $variables.foo is in $variables.foo", project)
	assert.Equal(t, "bar is in bar is in bar", expanded)
}

func TestExpandProjectUppercase(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("${variables.UPPERCASE}bar", project)
	assert.Equal(t, "foobar", expanded)
}
