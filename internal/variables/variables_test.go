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
  - name: foo-dashed
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
scripts:
  - name: test
    value: make test
  - name: recursive
    value: $scripts.recursive
  `)

	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	return project
}

func TestExpandProjectPlatformOs(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$platform.os", project)
	assert.NoError(t, Failure().ToError(), "Ran without failure")

	if runtime.GOOS != "darwin" {
		assert.Equal(t, runtime.GOOS, expanded, "Expanded platform variable")
	} else {
		assert.Equal(t, "macos", expanded, "Expanded platform variable")
	}
}

func TestExpandProjectHook(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$hooks.pre", project)
	assert.NoError(t, Failure().ToError(), "Ran without failure")
	assert.Equal(t, "echo 'Hello bar!'", expanded, "Expanded simple variable")
}

func TestExpandProjectHookWithConstraints(t *testing.T) {
	project := loadProject(t)

	if runtime.GOOS == "linux" {
		expanded := ExpandFromProject("$hooks.post", project)
		assert.NoError(t, Failure().ToError(), "Ran without failure")
		assert.Equal(t, "echo 'Hello baz!'", expanded, "Expanded platform-specific variable")
	} else if runtime.GOOS == "windows" {
		expanded := ExpandFromProject("$hooks.post", project)
		assert.NoError(t, Failure().ToError(), "Ran without failure")
		assert.Equal(t, "echo 'Hello quux!'", expanded, "Expanded platform-specific variable")
	}
}

func TestExpandProjectScript(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$ $scripts.test", project)
	assert.NoError(t, Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ make test", expanded, "Expanded simple script")
}

func TestExpandProjectAlternateSyntax(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("${platform.os}", project)
	assert.NoError(t, Failure().ToError(), "Ran without failure")
	if runtime.GOOS != "darwin" {
		assert.Equal(t, runtime.GOOS, expanded, "Expanded platform variable")
	} else {
		assert.Equal(t, "macos", expanded, "Expanded platform variable")
	}
}

func TestExpandProjectUnknownCategory(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$unknown.unknown", project)
	assert.Error(t, Failure().ToError(), "Ran with failure")
	assert.Equal(t, "", expanded, "Failed to expand")
	assert.True(t, Failure().Type.Matches(FailExpandVariableBadCategory), "Handled unknown category")
}

func TestExpandProjectUnknownName(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$platform.unknown", project)
	assert.Error(t, Failure().ToError(), "Ran with failure")
	assert.Equal(t, "", expanded, "Failed to expand")
	assert.True(t, Failure().Type.Matches(FailExpandVariableBadName), "Handled unknown category")
}

func TestExpandProjectInfiniteRecursion(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$scripts.recursive", project)
	assert.Error(t, Failure().ToError(), "Ran with failure")
	assert.Equal(t, "", expanded, "Failed to expand")
	assert.True(t, Failure().Type.Matches(FailExpandVariableRecursion), "Handled unknown category")
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
		ExpandFromProject(fmt.Sprintf("$platform.%s", name), project)
		assert.NoError(t, Failure().ToError(), "Ran without failure")
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
	assert.NoError(t, Failure().ToError(), "Ran without failure")
	assert.Equal(t, "bar is in bar is in bar", expanded)
}

func TestExpandProjectUppercase(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("${variables.UPPERCASE}bar", project)
	assert.NoError(t, Failure().ToError(), "Ran without failure")
	assert.Equal(t, "foobar", expanded)
}

func TestExpandDashed(t *testing.T) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
  variables:
    - name: foo-bar
      value: bar
  `)

	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	expanded := ExpandFromProject("- $variables.foo-bar -", project)
	assert.NoError(t, Failure().ToError(), "Ran without failure")
	assert.Equal(t, "- bar -", expanded)
}

func TestRegisterExpander_RequiresNonBlankName(t *testing.T) {
	failure := RegisterExpander("", func(n string, p *projectfile.Project) (string, *failures.Failure) {
		return "", nil
	})
	assert.True(t, failure.Type.Matches(FailExpanderBadName))
	assert.NotContains(t, expanderRegistry, "")

	failure = RegisterExpander(" \n \t\f ", func(n string, p *projectfile.Project) (string, *failures.Failure) {
		return "", nil
	})
	assert.True(t, failure.Type.Matches(FailExpanderBadName))
	assert.NotContains(t, expanderRegistry, " \n \t\f ")
}

func TestRegisterExpander_ExpanderFuncCannotBeNil(t *testing.T) {
	failure := RegisterExpander("tests", nil)
	assert.True(t, failure.Type.Matches(FailExpanderNoFunc))
	assert.NotContains(t, expanderRegistry, "")
}

func TestRegisterExpander(t *testing.T) {
	assert.NotContains(t, expanderRegistry, "lobsters")
	RegisterExpander("lobsters", func(n string, p *projectfile.Project) (string, *failures.Failure) {
		return "", nil
	})
	assert.Contains(t, expanderRegistry, "lobsters")
}
