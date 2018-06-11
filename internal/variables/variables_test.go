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
    os: darwin
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

	expanded, fail := ExpandFromProject("$platform.os", project)
	assert.Nil(t, fail, "Expanded without failure")
	assert.Equal(t, runtime.GOOS, expanded, "Expanded platform variable")
}

func TestExpandProjectHook(t *testing.T) {
	project := loadProject(t)

	expanded, fail := ExpandFromProject("$hooks.pre", project)
	assert.Nil(t, fail, "Expanded without failure")
	assert.Equal(t, "echo 'Hello bar!'", expanded, "Expanded simple variable")
}

func TestExpandProjectHookWithConstraints(t *testing.T) {
	project := loadProject(t)

	if runtime.GOOS == "linux" {
		expanded, fail := ExpandFromProject("$hooks.post", project)
		assert.Nil(t, fail, "Expanded without failure")
		assert.Equal(t, "echo 'Hello baz!'", expanded, "Expanded platform-specific variable")
	} else if runtime.GOOS == "windows" {
		expanded, fail := ExpandFromProject("$hooks.post", project)
		assert.Nil(t, fail, "Expanded without failure")
		assert.Equal(t, "echo 'Hello quux!'", expanded, "Expanded platform-specific variable")
	}
}

func TestExpandProjectCommand(t *testing.T) {
	project := loadProject(t)

	expanded, fail := ExpandFromProject("$ $commands.test", project)
	assert.Nil(t, fail, "Expanded without failure")
	assert.Equal(t, "$ make test", expanded, "Expanded simple command")
}

func TestExpandProjectAlternateSyntax(t *testing.T) {
	project := loadProject(t)

	expanded, fail := ExpandFromProject("${platform.os}", project)
	assert.Nil(t, fail, "Expanded without failure")
	assert.Equal(t, runtime.GOOS, expanded, "Expanded platform variable")
}

func TestExpandProjectUnknownCategory(t *testing.T) {
	project := loadProject(t)

	_, fail := ExpandFromProject("$unknown.unknown", project)
	assert.Error(t, fail.ToError(), "Error during expansion")
	assert.True(t, fail.Type.Matches(FailExpandVariableBadCategory), "Handled unknown category")
}

func TestExpandProjectUnknownName(t *testing.T) {
	project := loadProject(t)

	_, fail := ExpandFromProject("$platform.unknown", project)
	assert.Error(t, fail.ToError(), "Error during expansion")
	assert.True(t, fail.Type.Matches(FailExpandVariableBadName), "Handled unknown name")
}

func TestExpandProjectInfiniteRecursion(t *testing.T) {
	project := loadProject(t)

	_, fail := ExpandFromProject("$commands.recursive", project)
	assert.Error(t, fail.ToError(), "Error during expansion")
	assert.True(t, fail.Type.Matches(FailExpandVariableRecursion), "Handled infinite recursion")
}

// Tests all possible $platform.[name] variable expansions.
func TestExpandProjectPlatform(t *testing.T) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
  platforms:
    - name: Any
  `)

	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	for _, name := range []string{"name", "os", "version", "architecture", "libc", "compiler"} {
		_, fail := ExpandFromProject(fmt.Sprintf("$platform.%s", name), project)
		assert.Nil(t, fail, "Expanded without failure")
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

	expanded, fail := ExpandFromProject("$variables.foo is in $variables.foo is in $variables.foo", project)
	assert.Nil(t, fail, "Expanded without failure")
	assert.Equal(t, "bar is in bar is in bar", expanded)
}
