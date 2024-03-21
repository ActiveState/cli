package project_test

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"

	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func loadProject(t *testing.T) *project.Project {
	pjFile := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/Expander/general?branch=main"
lock: branchname@0.0.0-SHA123abcd
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
    value: echo $scripts.recursive
  - name: pythonScript
    language: python3
    value: scriptValue
  - name: scriptPath
    value: $scripts.pythonScript.path()
  - name: scriptRecursive
    value: $scripts.scriptRecursive.path()
  - name: bashScriptPath
    language: bash
    value: ${scripts.pythonScript.path()}
`)

	err := yaml.Unmarshal([]byte(contents), pjFile)
	assert.Nil(t, err, "Unmarshalled YAML")

	require.NoError(t, pjFile.Init())

	proj, err := project.FromWD()
	require.NoError(t, err)

	return proj
}

func TestExpandProject(t *testing.T) {
	prj := loadProject(t)
	prj.Source().SetPath(fmt.Sprintf("spoofed path%sactivestate.yaml", string(os.PathSeparator)))

	expanded, err := project.ExpandFromProject("$project.url()", prj)
	require.NoError(t, err)
	assert.Equal(t, prj.URL(), expanded)

	expanded, err = project.ExpandFromProject("$project.branch()", prj)
	require.NoError(t, err)
	assert.Equal(t, "main", expanded)

	expanded, err = project.ExpandFromProject("$project.owner()", prj)
	require.NoError(t, err)
	assert.Equal(t, "Expander", expanded)

	expanded, err = project.ExpandFromProject("$project.name()", prj)
	require.NoError(t, err)
	assert.Equal(t, "general", expanded)

	expanded, err = project.ExpandFromProject("$project.namespace()", prj)
	require.NoError(t, err)
	assert.Equal(t, "Expander/general", expanded)

	expanded, err = project.ExpandFromProject("$project.path()", prj)
	require.NoError(t, err)
	assert.Equal(t, "spoofed path", expanded)

	if runtime.GOOS == "windows" {
		prj.Source().SetPath(fmt.Sprintf(`c:\another\spoofed path\activestate.yaml`))
		expanded, err = project.ExpandFromProjectBashifyPaths("$project.path()", prj)
		require.NoError(t, err)
		assert.Equal(t, `/c/another/spoofed\ path`, expanded)
	}
}

func TestExpandTopLevel(t *testing.T) {
	prj := loadProject(t)

	expanded, err := project.ExpandFromProject("$project", prj)
	assert.NoError(t, err, "Ran without failure")

	assert.Equal(t, "https://platform.activestate.com/Expander/general?branch=main", expanded)

	expanded, err = project.ExpandFromProject("$lock", prj)
	assert.NoError(t, err, "Ran without failure")
	assert.Equal(t, "branchname@0.0.0-SHA123abcd", expanded)

	expanded, err = project.ExpandFromProject("$notcovered", prj)
	assert.NoError(t, err, "Ran without failure")
	assert.Equal(t, "$notcovered", expanded)
}

func TestExpandProjectScript(t *testing.T) {
	prj := loadProject(t)

	expanded, err := project.ExpandFromProject("$ $scripts.test", prj)
	assert.NoError(t, err, "Ran without failure")
	assert.Equal(t, "$ make test", expanded, "Expanded simple script")
}

func TestExpandProjectConstant(t *testing.T) {
	prj := loadProject(t)

	expanded, err := project.ExpandFromProject("$ $constants.constant", prj)
	assert.NoError(t, err, "Ran without failure")
	assert.Equal(t, "$ value", expanded, "Expanded simple constant")

	expanded, err = project.ExpandFromProject("$ $constants.recursive", prj)
	assert.NoError(t, err, "Ran without failure")
	assert.Equal(t, "$ recursive value", expanded, "Expanded recursive constant")
}

func TestExpandProjectSecret(t *testing.T) {
	pj := loadProject(t)

	project.RegisterExpander("secrets", func(_ string, category string, meta string, isFunction bool, ctx *project.Expansion) (string, error) {
		if category == project.ProjectCategory {
			return "proj-value", nil
		}
		return "user-proj-value", nil
	})

	expanded, err := project.ExpandFromProject("$ $secrets.user.user-proj-secret", pj)
	assert.NoError(t, err, "Ran without failure")
	assert.Equal(t, "$ user-proj-value", expanded, "Expanded simple constant")

	expanded, err = project.ExpandFromProject("$ $secrets.project.proj-secret", pj)
	assert.NoError(t, err, "Ran without failure")
	assert.Equal(t, "$ proj-value", expanded, "Expanded simple constant")
}

func TestExpandProjectAlternateSyntax(t *testing.T) {
	prj := loadProject(t)

	expanded, err := project.ExpandFromProject("${project.name()}", prj)
	assert.NoError(t, err, "Ran without failure")
	assert.Equal(t, "general", expanded, "Expanded project variable")
}

func TestExpandProjectUnknownCategory(t *testing.T) {
	prj := loadProject(t)

	expanded, err := project.ExpandFromProject("$unknown.unknown", prj)
	assert.NoError(t, err, "Ran without failure")
	assert.Equal(t, "$unknown.unknown", expanded, "Didn't expand variable it doesnt own")
}

func TestExpandProjectInfiniteRecursion(t *testing.T) {
	prj := loadProject(t)

	_, err := project.ExpandFromProject("$scripts.recursive", prj)
	require.Error(t, err, "Ran with failure")
	assert.Contains(t, err.Error(), "Infinite recursion trying to expand variable", "Handled unknown category")
}

func TestExpandDashed(t *testing.T) {
	projectFile := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/Expander/Dashed"
scripts:
  - name: foo-bar
    value: bar
`)

	err := yaml.Unmarshal([]byte(contents), projectFile)
	assert.Nil(t, err, "Unmarshalled YAML")
	prj, err := project.FromWD()
	require.NoError(t, err)

	expanded, err := project.ExpandFromProject("- $scripts.foo-bar -", prj)
	assert.NoError(t, err, "Ran without failure")
	assert.Equal(t, "- bar -", expanded)
}

func TestExpandScriptPath(t *testing.T) {
	prj := loadProject(t)

	expanded, err := project.ExpandFromProject("$scripts.scriptPath", prj)
	assert.NoError(t, err, "Ran without failure")
	assert.True(t, strings.HasSuffix(expanded, language.Python3.Ext()), fmt.Sprintf("%s should have suffix %s", expanded, language.Python3.Ext()))

	contents, err := fileutils.ReadFile(expanded)
	require.NoError(t, err)
	assert.Contains(t, string(contents), language.Python3.Header(), "Has Python3 header")
	assert.Contains(t, string(contents), "scriptValue", "Contains intended script value")
}

func TestExpandScriptPathRecursive(t *testing.T) {
	prj := loadProject(t)

	expanded, err := project.ExpandFromProject("$scripts.scriptRecursive", prj)
	assert.NoError(t, err, "Ran without failure")

	contents, err := fileutils.ReadFile(expanded)
	require.NoError(t, err)
	assert.NotContains(t, contents, "$scripts.scriptRecursive.path()")
}

func TestExpandBashScriptPath(t *testing.T) {
	prj := loadProject(t)
	script := prj.ScriptByName("bashScriptPath")
	require.NotNil(t, script, "bashScriptPath script does not exist")
	value, err := script.Value()
	require.NoError(t, err)
	assert.Contains(t, value, "/pythonScript") // assert bash backslashes, even on Windows
}
