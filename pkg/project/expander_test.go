package project_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/language"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func loadProject(t *testing.T) *project.Project {
	projectfile.Reset()

	pjFile := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/Expander/general?commitID=00010001-0001-0001-0001-000100010001"
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
  - name: pythonScript
    languages: [python3]
    value: scriptValue
  - name: scriptPath
    value: $scripts.pythonScript.path()
  - name: scriptRecursive
    value: $scripts.scriptRecursive.path()
`)

	err := yaml.Unmarshal([]byte(contents), pjFile)
	assert.Nil(t, err, "Unmarshalled YAML")

	pjFile.Persist()

	return project.Get()
}

func TestExpandProjectPlatformOs(t *testing.T) {
	prj := loadProject(t)

	expanded := project.ExpandFromProject("$platform.os", prj)
	assert.NoError(t, project.Failure().ToError(), "Ran without failure")

	if runtime.GOOS != "darwin" {
		assert.Equal(t, runtime.GOOS, expanded, "Expanded platform variable")
	} else {
		assert.Equal(t, "macos", expanded, "Expanded platform variable")
	}
}

func TestExpandProjectScript(t *testing.T) {
	prj := loadProject(t)

	expanded := project.ExpandFromProject("$ $scripts.test", prj)
	assert.NoError(t, project.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ make test", expanded, "Expanded simple script")
}

func TestExpandProjectConstant(t *testing.T) {
	prj := loadProject(t)

	expanded := project.ExpandFromProject("$ $constants.constant", prj)
	assert.NoError(t, project.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ value", expanded, "Expanded simple constant")

	expanded = project.ExpandFromProject("$ $constants.recursive", prj)
	assert.NoError(t, project.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ recursive value", expanded, "Expanded recursive constant")
}

func TestExpandProjectSecret(t *testing.T) {
	pj := loadProject(t)

	project.RegisterExpander("secrets", func(category string, meta string, isFunction bool, pj *project.Project) (string, *failures.Failure) {
		if category == project.ProjectCategory {
			return "proj-value", nil
		}
		return "user-proj-value", nil
	})

	expanded := project.ExpandFromProject("$ $secrets.user.user-proj-secret", pj)
	assert.NoError(t, project.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ user-proj-value", expanded, "Expanded simple constant")

	expanded = project.ExpandFromProject("$ $secrets.project.proj-secret", pj)
	assert.NoError(t, project.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "$ proj-value", expanded, "Expanded simple constant")
}

func TestExpandProjectAlternateSyntax(t *testing.T) {
	prj := loadProject(t)

	expanded := project.ExpandFromProject("${platform.os}", prj)
	assert.NoError(t, project.Failure().ToError(), "Ran without failure")
	if runtime.GOOS != "darwin" {
		assert.Equal(t, runtime.GOOS, expanded, "Expanded platform variable")
	} else {
		assert.Equal(t, "macos", expanded, "Expanded platform variable")
	}
}

func TestExpandProjectUnknownCategory(t *testing.T) {
	prj := loadProject(t)

	expanded := project.ExpandFromProject("$unknown.unknown", prj)
	assert.Error(t, project.Failure().ToError(), "Ran with failure")
	assert.Equal(t, "", expanded, "Failed to expand")
	assert.True(t, project.Failure().Type.Matches(project.FailExpandVariableBadCategory), "Handled unknown category")
}

func TestExpandProjectUnknownName(t *testing.T) {
	prj := loadProject(t)

	expanded := project.ExpandFromProject("$platform.unknown", prj)
	assert.Error(t, project.Failure().ToError(), "Ran with failure")
	assert.Equal(t, "", expanded, "Failed to expand")
	assert.True(t, project.Failure().Type.Matches(project.FailExpandVariableBadName), "Handled unknown category")
}

func TestExpandProjectInfiniteRecursion(t *testing.T) {
	prj := loadProject(t)

	expanded := project.ExpandFromProject("$scripts.recursive", prj)
	require.Error(t, project.Failure().ToError(), "Ran with failure")
	assert.Equal(t, "", expanded, "Failed to expand")
	assert.True(t, project.Failure().Type.Matches(project.FailExpandVariableRecursion), "Handled unknown category")
}

// Tests all possible $platform.[name] variable expansions.
func TestExpandProjectPlatform(t *testing.T) {
	projectFile := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: https://platform.activestate.com/Expander/Plarforms?commitID=00010001-0001-0001-0001-000100010001"
platforms:
  - name: Any
`)

	err := yaml.Unmarshal([]byte(contents), projectFile)
	assert.Nil(t, err, "Unmarshalled YAML")
	projectFile.Persist()
	prj := project.Get()

	for _, name := range []string{"name", "os", "version", "architecture", "libc", "compiler"} {
		project.ExpandFromProject(fmt.Sprintf("$platform.%s", name), prj)
		assert.NoError(t, project.Failure().ToError(), "Ran without failure")
	}
}

func TestExpandDashed(t *testing.T) {
	projectFile := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/Expander/Dashed?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: foo-bar
    value: bar
`)

	err := yaml.Unmarshal([]byte(contents), projectFile)
	assert.Nil(t, err, "Unmarshalled YAML")
	projectFile.Persist()
	prj := project.Get()

	expanded := project.ExpandFromProject("- $scripts.foo-bar -", prj)
	assert.NoError(t, project.Failure().ToError(), "Ran without failure")
	assert.Equal(t, "- bar -", expanded)
	projectfile.Reset()
}

func TestExpandScriptPath(t *testing.T) {
	prj := loadProject(t)

	expanded := project.ExpandFromProject("$scripts.scriptPath", prj)
	assert.NoError(t, project.Failure().ToError(), "Ran without failure")
	assert.True(t, strings.HasSuffix(expanded, language.Python3.Ext()), fmt.Sprintf("%s should have suffix %s", expanded, language.Python3.Ext()))

	contents, fail := fileutils.ReadFile(expanded)
	require.NoError(t, fail.ToError())
	assert.Contains(t, string(contents), language.Python3.Header(), "Has Python3 header")
	assert.Contains(t, string(contents), "scriptValue", "Contains intended script value")
}

func TestExpandScriptPathRecursive(t *testing.T) {
	prj := loadProject(t)

	expanded := project.ExpandFromProject("$scripts.scriptRecursive", prj)
	assert.NoError(t, project.Failure().ToError(), "Ran without failure")

	contents, fail := fileutils.ReadFile(expanded)
	require.NoError(t, fail.ToError())
	assert.NotContains(t, contents, "$scripts.scriptRecursive.path()")
}
