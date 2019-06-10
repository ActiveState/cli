package project

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"

	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func init() {
	secretsClient := secretsapi.NewDefaultClient(authentication.Get().BearerToken())
	RegisterExpander("variables", NewVarPromptingExpander(secretsClient))
}

func loadProject(t *testing.T) *Project {
	projectfile.Reset()

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://https://platform.activestate.com/string/string/string"
platforms:
  - name: Linux
    os: linux
  - name: Windows
    os: windows
  - name: macOS
    os: macos
events:
  - name: pre
    value: echo 'Hello $variables.foo!'
  - name: post
    value: echo 'Hello $variables.bar!'
constants:
  - name: constant
    value: value
  - name: recursive
    value: recursive $constants.constant
scripts:
  - name: test
    value: make test
  - name: recursive
    value: $scripts.recursive
  `)

	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")

	fail := project.Parse()
	assert.NoError(t, fail.ToError())

	project.Persist()

	return New(project)
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

func TestExpandProjectScript(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$ $scripts.test", project)
	assert.NoError(t, Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ make test", expanded, "Expanded simple script")
}

func TestExpandProjectConstant(t *testing.T) {
	project := loadProject(t)

	expanded := ExpandFromProject("$ $constants.constant", project)
	assert.NoError(t, Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ value", expanded, "Expanded simple constant")

	expanded = ExpandFromProject("$ $constants.recursive", project)
	assert.NoError(t, Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ recursive value", expanded, "Expanded recursive constant")
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
	projectFile := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://https://platform.activestate.com/string/string/string"
platforms:
  - name: Any
  `)

	err := yaml.Unmarshal([]byte(contents), projectFile)
	assert.Nil(t, err, "Unmarshalled YAML")
	projectFile.Persist()
	project := New(projectFile)

	for _, name := range []string{"name", "os", "version", "architecture", "libc", "compiler"} {
		ExpandFromProject(fmt.Sprintf("$platform.%s", name), project)
		assert.NoError(t, Failure().ToError(), "Ran without failure")
	}
}

func TestExpandDashed(t *testing.T) {
	projectFile := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://https://platform.activestate.com/string/string/string"
scripts:
  - name: foo-bar
    value: bar
  `)

	err := yaml.Unmarshal([]byte(contents), projectFile)
	assert.Nil(t, err, "Unmarshalled YAML")
	fail := projectFile.Parse()
	assert.NoError(t, fail.ToError())
	projectFile.Persist()
	project := New(projectFile)

	expanded := ExpandFromProject("- $scripts.foo-bar -", project)
	assert.NoError(t, Failure().ToError(), "Ran without failure")
	assert.Equal(t, "- bar -", expanded)
}
