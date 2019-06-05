package expander_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/expander"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func init() {
	secretsClient := secretsapi.NewDefaultClient(authentication.Get().BearerToken())
	expander.RegisterExpander("variables", expander.NewVarPromptingExpander(secretsClient))
}

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
events:
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

	fail := project.Parse()
	assert.NoError(t, fail.ToError())

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
	fail := project.Parse()
	assert.NoError(t, fail.ToError())
	project.Persist()

	expanded := expander.ExpandFromProject("- $scripts.foo-bar -", project)
	assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "- bar -", expanded)
}
