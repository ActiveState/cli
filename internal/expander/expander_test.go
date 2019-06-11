package expander_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/expander"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func loadProject(t *testing.T) *projectfile.Project {
	projectfile.Reset()

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
name: string
owner: string
platforms:
  - name: Linux
    os: linux
  - name: Windows
    os: windows
  - name: macOS
    os: macos
constants:
  - name: constant
    value: value
  - name: recursive
    value: recursive $constants.constant
secrets:
  project:
    - name: proj-secret
  user:
    - name: user-proj-secret
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

	expanded := expander.ExpandFromProject("$platform.os", project)
	assert.NoError(t, expander.Failure().ToError(), "Ran without failure")

	if runtime.GOOS != "darwin" {
		assert.Equal(t, runtime.GOOS, expanded, "Expanded platform variable")
	} else {
		assert.Equal(t, "macos", expanded, "Expanded platform variable")
	}
}

func TestExpandProjectScript(t *testing.T) {
	project := loadProject(t)

	expanded := expander.ExpandFromProject("$ $scripts.test", project)
	assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ make test", expanded, "Expanded simple script")
}

func TestExpandProjectConstant(t *testing.T) {
	project := loadProject(t)

	expanded := expander.ExpandFromProject("$ $constants.constant", project)
	assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ value", expanded, "Expanded simple constant")

	expanded = expander.ExpandFromProject("$ $constants.recursive", project)
	assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ recursive value", expanded, "Expanded recursive constant")
}

func TestExpandProjectSecret(t *testing.T) {
	project := loadProject(t)

	expander.RegisterExpander("secrets.user", func(name string, project *projectfile.Project) (string, *failures.Failure) {
		return "user-proj-value", nil
	})

	expander.RegisterExpander("secrets.project", func(name string, project *projectfile.Project) (string, *failures.Failure) {
		return "proj-value", nil
	})

	expanded := expander.ExpandFromProject("$ $secrets.user.user-proj-secret", project)
	assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ user-proj-value", expanded, "Expanded simple constant")

	expanded = expander.ExpandFromProject("$ $secrets.project.proj-secret", project)
	assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ proj-value", expanded, "Expanded simple constant")
}

func TestExpandProjectAlternateSyntax(t *testing.T) {
	project := loadProject(t)

	expanded := expander.ExpandFromProject("${platform.os}", project)
	assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
	if runtime.GOOS != "darwin" {
		assert.Equal(t, runtime.GOOS, expanded, "Expanded platform variable")
	} else {
		assert.Equal(t, "macos", expanded, "Expanded platform variable")
	}
}

func TestExpandProjectUnknownCategory(t *testing.T) {
	project := loadProject(t)

	expanded := expander.ExpandFromProject("$unknown.unknown", project)
	assert.Error(t, expander.Failure().ToError(), "Ran with failure")
	assert.Equal(t, "", expanded, "Failed to expand")
	assert.True(t, expander.Failure().Type.Matches(expander.FailExpandVariableBadCategory), "Handled unknown category")
}

func TestExpandProjectUnknownName(t *testing.T) {
	project := loadProject(t)

	expanded := expander.ExpandFromProject("$platform.unknown", project)
	assert.Error(t, expander.Failure().ToError(), "Ran with failure")
	assert.Equal(t, "", expanded, "Failed to expand")
	assert.True(t, expander.Failure().Type.Matches(expander.FailExpandVariableBadName), "Handled unknown category")
}

func TestExpandProjectInfiniteRecursion(t *testing.T) {
	project := loadProject(t)

	expanded := expander.ExpandFromProject("$scripts.recursive", project)
	assert.Error(t, expander.Failure().ToError(), "Ran with failure")
	assert.Equal(t, "", expanded, "Failed to expand")
	assert.True(t, expander.Failure().Type.Matches(expander.FailExpandVariableRecursion), "Handled unknown category")
}

// Tests all possible $platform.[name] variable expansions.
func TestExpandProjectPlatform(t *testing.T) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
name: string
owner: string
platforms:
  - name: Any
  `)

	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	for _, name := range []string{"name", "os", "version", "architecture", "libc", "compiler"} {
		expander.ExpandFromProject(fmt.Sprintf("$platform.%s", name), project)
		assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
	}
}

func TestExpandDashed(t *testing.T) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
name: string
owner: string
scripts:
  - name: foo-bar
    value: bar
  `)

	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	expanded := expander.ExpandFromProject("- $scripts.foo-bar -", project)
	assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "- bar -", expanded)
}
