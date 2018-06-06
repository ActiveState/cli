package variables

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestExpandFromProject(t *testing.T) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
platforms:
  - name: Linux
    os: linux
  - name: Windows
    os: windows
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

	// Test various expansions.
	expanded, fail := ExpandFromProject("$platform.os", project)
	assert.Nil(t, fail, "Expanded without failure")
	assert.Equal(t, runtime.GOOS, expanded, "Expanded platform variable")
	expanded, fail = ExpandFromProject("$hooks.pre", project)
	assert.Nil(t, fail, "Expanded without failure")
	assert.Equal(t, "echo 'Hello bar!'", expanded, "Expanded simple variable")
	if runtime.GOOS == "linux" {
		expanded, fail = ExpandFromProject("$hooks.post", project)
		assert.Nil(t, fail, "Expanded without failure")
		assert.Equal(t, "echo 'Hello baz!'", expanded, "Expanded platform-specific variable")
	} else if runtime.GOOS == "windows" {
		expanded, fail = ExpandFromProject("$hooks.post", project)
		assert.Nil(t, fail, "Expanded without failure")
		assert.Equal(t, "echo 'Hello quux!'", expanded, "Expanded platform-specific variable")
	}
	expanded, fail = ExpandFromProject("$ $commands.test", project)
	assert.Nil(t, fail, "Expanded without failure")
	assert.Equal(t, "$ make test", expanded, "Expanded simple command")

	// Test alternate ${} syntax.
	expanded, fail = ExpandFromProject("${platform.os}", project)
	assert.Nil(t, fail, "Expanded without failure")
	assert.Equal(t, runtime.GOOS, expanded, "Expanded platform variable")

	// Test unknown category expansion failure.
	_, fail = ExpandFromProject("$unknown.unknown", project)
	assert.Error(t, fail.ToError(), "Error during expansion")
	assert.True(t, fail.Type.Matches(failures.FailExpandVariableBadCategory), "Handled unknown category")

	// Test unknown name expansion failure.
	_, fail = ExpandFromProject("$platform.unknown", project)
	assert.Error(t, fail.ToError(), "Error during expansion")
	assert.True(t, fail.Type.Matches(failures.FailExpandVariableBadName), "Handled unknown name")

	// Test infinite recursion failure.
	_, fail = ExpandFromProject("$commands.recursive", project)
	assert.Error(t, fail.ToError(), "Error during expansion")
	assert.True(t, fail.Type.Matches(failures.FailExpandVariableRecursion), "Handled infinite recursion")
}

// Tests all possible $platform.[name] variable expansions.
func TestExpandPlatformFromProject(t *testing.T) {
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
