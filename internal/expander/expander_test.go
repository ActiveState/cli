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

func TestExpandProjectEvent(t *testing.T) {
	project := loadProject(t)

	expanded := expander.ExpandFromProject("$events.pre", project)
	assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "echo 'Hello bar!'", expanded, "Expanded simple variable")
}

func TestExpandProjectEventWithConstraints(t *testing.T) {
	project := loadProject(t)

	if runtime.GOOS == "linux" {
		expanded := expander.ExpandFromProject("$events.post", project)
		assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
		assert.Equal(t, "echo 'Hello baz!'", expanded, "Expanded platform-specific variable")
	} else if runtime.GOOS == "windows" {
		expanded := expander.ExpandFromProject("$events.post", project)
		assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
		assert.Equal(t, "echo 'Hello quux!'", expanded, "Expanded platform-specific variable")
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

func TestExpandProjectEmbedded(t *testing.T) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
name: string
owner: string
variables:
  - name: foo
    value: bar
  `)

	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	fail := project.Parse()
	assert.NoError(t, fail.ToError())
	project.Persist()

	expanded := expander.ExpandFromProject("$variables.foo is in $variables.foo is in $variables.foo", project)
	assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "bar is in bar is in bar", expanded)
}

func TestExpandProjectUppercase(t *testing.T) {
	project := loadProject(t)

	expanded := expander.ExpandFromProject("${variables.UPPERCASE}bar", project)
	assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "foobar", expanded)
}

func TestExpandDashed(t *testing.T) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
name: string
owner: string
variables:
  - name: foo-bar
    value: bar
  `)

	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	fail := project.Parse()
	assert.NoError(t, fail.ToError())
	project.Persist()

	expanded := expander.ExpandFromProject("- $variables.foo-bar -", project)
	assert.NoError(t, expander.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "- bar -", expanded)
}
